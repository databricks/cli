package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	authcmd "github.com/databricks/cli/cmd/auth"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/databricks-sdk-go/service/provisioning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	workspaceTestAccountID = "test-account"
	workspaceTestToken     = "test-token"
	workspaceSkipLabel     = "Skip (account-level access only)"
	workspacePickerLabel   = "Select a workspace"
	workspaceIDPrompt      = "Enter workspace ID (empty to skip)"
)

func TestWorkspaceSearcher(t *testing.T) {
	items := []cmdio.Tuple{
		{Name: "Analytics", Id: "1234"},
		{Name: "Report 123", Id: "99"},
		{Name: "Production", Id: "123"},
		{Name: "Shared", Id: "887766"},
		{Name: "Shared", Id: "778899"},
		{Name: "", Id: "40"},
		{Name: "\u00c5ngstr\u00f6m", Id: "9007199254740993"},
		{Name: "Skip Analytics", Id: "0"},
		{Name: workspaceSkipLabel, Id: ""},
	}
	search := authcmd.NewWorkspaceSearcher(items)
	for _, tc := range []struct {
		name  string
		query string
		want  []int
	}{
		{name: "empty", want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8}},
		{name: "whitespace", query: " \t ", want: []int{0, 1, 2, 3, 4, 5, 6, 7, 8}},
		{name: "name", query: "aNaLyT", want: []int{0, 7}},
		{name: "trim name", query: "  Production  ", want: []int{2}},
		{name: "partial id", query: "887", want: []int{3}},
		{name: "partial name or id", query: "12", want: []int{0, 1, 2}},
		{name: "exact id beats longer id and numeric name", query: "123", want: []int{2}},
		{name: "trim exact id", query: "\t123 ", want: []int{2}},
		{name: "duplicate names", query: "shared", want: []int{3, 4}},
		{name: "duplicate name by id", query: "778899", want: []int{4}},
		{name: "empty name", query: "40", want: []int{5}},
		{name: "non ascii name", query: "\u00e5NGSTR\u00d6M", want: []int{6}},
		{name: "large id", query: "9007199254740993", want: []int{6}},
		{name: "zero id is not skip", query: "0", want: []int{7}},
		{name: "skip stays last", query: "sKiP", want: []int{7, 8}},
		{name: "skip by label", query: "account-level", want: []int{8}},
		{name: "missing name", query: "not-a-workspace"},
		{name: "missing id", query: "555555"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var matches []int
			for i := range items {
				if search(tc.query, i) {
					matches = append(matches, i)
				}
			}
			assert.Equal(t, tc.want, matches)
		})
	}
}

func TestPromptForWorkspaceSelection(t *testing.T) {
	workspaces := make([]provisioning.Workspace, 60)
	for i := range workspaces {
		workspaces[i] = provisioning.Workspace{
			WorkspaceId:   int64(1000 + i),
			WorkspaceName: fmt.Sprintf("Workspace %c%c", 'A'+i/26, 'A'+i%26),
		}
	}
	workspaces[59] = provisioning.Workspace{WorkspaceId: 9007199254740993, WorkspaceName: "Target Workspace"}

	for _, tc := range []struct {
		name       string
		workspaces []provisioning.Workspace
		input      string
		want       string
	}{
		{name: "id beyond first fifty", workspaces: workspaces, input: "9007199254740993", want: "9007199254740993"},
		{name: "name beyond first fifty", workspaces: workspaces, input: "tArGeT", want: "9007199254740993"},
		{
			name: "exact id precedence",
			workspaces: []provisioning.Workspace{
				{WorkspaceId: 1234, WorkspaceName: "Longer ID"},
				{WorkspaceId: 99, WorkspaceName: "Report 123"},
				{WorkspaceId: 123, WorkspaceName: "Exact ID"},
			},
			input: " 123 ", want: "123",
		},
		{
			name: "duplicate names",
			workspaces: []provisioning.Workspace{
				{WorkspaceId: 111, WorkspaceName: "Shared"},
				{WorkspaceId: 222, WorkspaceName: "Shared"},
			},
			input: "222", want: "222",
		},
		{
			name: "empty name",
			workspaces: []provisioning.Workspace{
				{WorkspaceId: 111, WorkspaceName: "Named"},
				{WorkspaceId: 222},
			},
			input: "222", want: "222",
		},
		{
			name: "workspace before skip",
			workspaces: []provisioning.Workspace{
				{WorkspaceId: 111, WorkspaceName: "Named"},
				{WorkspaceId: 222, WorkspaceName: "Skip Analytics"},
			},
			input: "skip", want: "222",
		},
		{name: "skip", workspaces: workspaces, input: "skip", want: ""},
		{name: "whitespace preserves API order", workspaces: workspaces, input: "  ", want: "1000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, requests := newWorkspaceListServer(t, http.StatusOK, tc.workspaces)
			tm := termtest.Run(t, func(ctx context.Context) (string, error) {
				return authcmd.PromptForWorkspaceSelection(ctx, args, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: workspaceTestToken}))
			})
			tm.WaitFor(workspacePickerLabel)
			tm.Type(tc.input + termtest.KeyEnter)
			id, err := tm.Result()
			require.NoError(t, err)
			assert.Equal(t, tc.want, id)
			assert.EqualValues(t, 1, requests.Load())
			assert.NotContains(t, tm.Raw(), workspaceIDPrompt)
			assert.NotContains(t, tm.Raw(), "Showing first")
		})
	}
}

func TestPromptForWorkspaceSelectionNoResults(t *testing.T) {
	args, requests := newWorkspaceListServer(t, http.StatusOK, []provisioning.Workspace{
		{WorkspaceId: 111, WorkspaceName: "Alpha"},
		{WorkspaceId: 222, WorkspaceName: "Beta"},
		{WorkspaceId: 333},
	})
	tm := termtest.Run(t, func(ctx context.Context) (string, error) {
		return authcmd.PromptForWorkspaceSelection(ctx, args, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: workspaceTestToken}))
	})
	tm.WaitFor(workspacePickerLabel)
	tm.Golden("01-initial")

	tm.Type("999999")
	tm.WaitFor("No results")
	tm.Type(termtest.KeyEnter)
	tm.Type(strings.Repeat(termtest.KeyBackspace, 6))
	require.Eventually(t, func() bool {
		return strings.Contains(tm.Snapshot(), workspaceSkipLabel) && !strings.Contains(tm.Snapshot(), "No results")
	}, 5*time.Second, 5*time.Millisecond)
	tm.Type(termtest.KeyDown + termtest.KeyEnter)
	id, err := tm.Result()
	require.NoError(t, err)
	assert.Equal(t, "222", id)
	assert.EqualValues(t, 1, requests.Load())
}

func TestPromptForWorkspaceSelectionWithoutPicker(t *testing.T) {
	for _, tc := range []struct {
		name       string
		workspaces []provisioning.Workspace
		want       string
		message    string
	}{
		{name: "no workspaces", workspaces: []provisioning.Workspace{}},
		{
			name:       "one workspace",
			workspaces: []provisioning.Workspace{{WorkspaceId: 123, WorkspaceName: "Only Workspace"}},
			want:       "123",
			message:    `Auto-selected workspace "Only Workspace" (123)`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, requests := newWorkspaceListServer(t, http.StatusOK, tc.workspaces)
			var output bytes.Buffer
			ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader(""), io.Discard, &output))
			id, err := authcmd.PromptForWorkspaceSelection(ctx, args, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: workspaceTestToken}))
			require.NoError(t, err)
			assert.Equal(t, tc.want, id)
			assert.EqualValues(t, 1, requests.Load())
			assert.Contains(t, output.String(), tc.message)
			assert.NotContains(t, output.String(), workspacePickerLabel)
			assert.NotContains(t, output.String(), workspaceIDPrompt)
		})
	}
}

func TestPromptForWorkspaceSelectionNonInteractive(t *testing.T) {
	args, requests := newWorkspaceListServer(t, http.StatusOK, nil)
	ctx, output := cmdio.NewTestContextWithStderr(t.Context())
	id, err := authcmd.PromptForWorkspaceSelection(ctx, args, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: workspaceTestToken}))
	require.NoError(t, err)
	assert.Empty(t, id)
	assert.Zero(t, requests.Load())
	assert.Contains(t, output.String(), "set workspace_id in your profile")
}

func TestPromptForWorkspaceSelectionListError(t *testing.T) {
	for _, input := range []string{" 123456 ", ""} {
		t.Run(input, func(t *testing.T) {
			args, requests := newWorkspaceListServer(t, http.StatusForbidden, nil)
			tm := termtest.Run(t, func(ctx context.Context) (string, error) {
				return authcmd.PromptForWorkspaceSelection(ctx, args, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: workspaceTestToken}))
			})
			tm.WaitFor(workspaceIDPrompt)
			tm.Type(input + termtest.KeyEnter)
			id, err := tm.Result()
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(input), id)
			assert.EqualValues(t, 1, requests.Load())
			assert.NotContains(t, tm.Raw(), workspacePickerLabel)
		})
	}
}

// newWorkspaceListServer exercises the SDK's actual list decoding without a live account.
func newWorkspaceListServer(t *testing.T, status int, workspaces []provisioning.Workspace) (*auth.AuthArguments, *atomic.Int32) {
	t.Helper()
	// The SDK client reads DATABRICKS_* variables and ~/.databrickscfg, so a
	// developer's DATABRICKS_CONFIG_PROFILE would otherwise break client setup.
	testutil.CleanupEnvironment(t)
	body, err := json.Marshal(workspaces)
	require.NoError(t, err)
	if status != http.StatusOK {
		body = []byte(`{"error_code":"PERMISSION_DENIED","message":"Account access denied"}`)
	}
	requests := new(atomic.Int32)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/2.0/accounts/"+workspaceTestAccountID+"/workspaces", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Empty(t, r.URL.RawQuery)
		assert.Equal(t, "Bearer "+workspaceTestToken, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, err := w.Write(body)
		assert.NoError(t, err)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &auth.AuthArguments{Host: server.URL, AccountID: workspaceTestAccountID}, requests
}
