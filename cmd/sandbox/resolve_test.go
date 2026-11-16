package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveLocalIDEmptyArgIsError(t *testing.T) {
	ctx, _ := stateCtx(t)
	_, err := resolveLocalID(ctx, "p", "")
	require.Error(t, err)
}

func TestResolveLocalIDPassThroughWhenCacheEmpty(t *testing.T) {
	ctx, _ := stateCtx(t)
	id, err := resolveLocalID(ctx, "p", "anything-goes")
	require.NoError(t, err)
	assert.Equal(t, "anything-goes", id)
}

func TestResolveLocalIDIDFirstPrecedence(t *testing.T) {
	// A sandbox whose --name happens to collide with another sandbox's ID
	// must NOT resolve the typed ID via the name path.
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{
		{ID: "happy-panda-1234", Name: "real-panda"},
		{ID: "other-id-5678", Name: "happy-panda-1234"},
	}))
	id, err := resolveLocalID(ctx, "p", "happy-panda-1234")
	require.NoError(t, err)
	assert.Equal(t, "happy-panda-1234", id, "must resolve to the ID, not the entry that has this string as its --name")
}

func TestResolveLocalIDNameMatch(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{
		{ID: "happy-panda-1234", Name: "my-project"},
	}))
	id, err := resolveLocalID(ctx, "p", "my-project")
	require.NoError(t, err)
	assert.Equal(t, "happy-panda-1234", id)
}

func TestResolveLocalIDAmbiguousNameErrors(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{
		{ID: "happy-panda-1234", Name: "foo"},
		{ID: "sad-otter-5678", Name: "foo"},
	}))
	_, err := resolveLocalID(ctx, "p", "foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
	assert.Contains(t, err.Error(), "happy-panda-1234")
	assert.Contains(t, err.Error(), "sad-otter-5678")
}

func TestResolveLocalIDEmptyNamesIgnored(t *testing.T) {
	// Sandboxes without a --name should not match an empty-string arg or
	// any other arg by virtue of their name being empty.
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{
		{ID: "happy-panda-1234", Name: ""},
	}))
	_, err := resolveLocalID(ctx, "p", "")
	require.Error(t, err, "empty arg always errors")

	id, err := resolveLocalID(ctx, "p", "happy-panda-1234")
	require.NoError(t, err)
	assert.Equal(t, "happy-panda-1234", id)
}

func TestSandboxesPerProfileIsolated(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p-a", []cachedSandbox{{ID: "a-1", Name: "shared-name"}}))
	require.NoError(t, setSandboxes(ctx, "p-b", []cachedSandbox{{ID: "b-1", Name: "shared-name"}}))

	id, err := resolveLocalID(ctx, "p-a", "shared-name")
	require.NoError(t, err)
	assert.Equal(t, "a-1", id)

	id, err = resolveLocalID(ctx, "p-b", "shared-name")
	require.NoError(t, err)
	assert.Equal(t, "b-1", id)
}

func TestUpsertSandboxAddsNewEntry(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, upsertSandbox(ctx, "p", "id-1", "name-1"))
	assert.Equal(t, []cachedSandbox{{ID: "id-1", Name: "name-1"}}, getSandboxes(ctx, "p"))
}

func TestUpsertSandboxUpdatesExistingEntry(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, upsertSandbox(ctx, "p", "id-1", "old"))
	require.NoError(t, upsertSandbox(ctx, "p", "id-1", "new"))
	assert.Equal(t, []cachedSandbox{{ID: "id-1", Name: "new"}}, getSandboxes(ctx, "p"))
}

func TestUpsertSandboxSameValueIsNoop(t *testing.T) {
	ctx, path := stateCtx(t)
	require.NoError(t, upsertSandbox(ctx, "p", "id-1", "name-1"))
	before, err := os.Stat(path)
	require.NoError(t, err)
	require.NoError(t, upsertSandbox(ctx, "p", "id-1", "name-1"))
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), after.ModTime(), "no-op upsert must not rewrite the file")
}

func TestRemoveSandboxDropsEntry(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{
		{ID: "keep-me", Name: ""},
		{ID: "drop-me", Name: "doomed"},
	}))
	require.NoError(t, removeSandbox(ctx, "p", "drop-me"))
	assert.Equal(t, []cachedSandbox{{ID: "keep-me", Name: ""}}, getSandboxes(ctx, "p"))
}

func TestRemoveSandboxMissingIsNoop(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{{ID: "keep-me"}}))
	require.NoError(t, removeSandbox(ctx, "p", "nope"))
	assert.Equal(t, []cachedSandbox{{ID: "keep-me"}}, getSandboxes(ctx, "p"))
}

// Removing the last sandbox for a profile must also drop the profile's
// cached gateway host — otherwise sandbox.json accumulates orphan
// gatewayHosts entries that no longer correspond to any sandbox.
func TestRemoveSandboxClearsOrphanGatewayHost(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{{ID: "only-one"}}))
	require.NoError(t, setGatewayHost(ctx, "p", "gw.example.test"))
	require.Equal(t, "gw.example.test", getGatewayHost(ctx, "p"))

	require.NoError(t, removeSandbox(ctx, "p", "only-one"))
	assert.Empty(t, getSandboxes(ctx, "p"))
	assert.Empty(t, getGatewayHost(ctx, "p"), "gateway host must be cleared when the last sandbox is removed")
}

// Removing one of many sandboxes must NOT touch the gateway host — it
// still applies to the remaining sandboxes on the profile.
func TestRemoveSandboxKeepsGatewayHostWhileSandboxesRemain(t *testing.T) {
	ctx, _ := stateCtx(t)
	require.NoError(t, setSandboxes(ctx, "p", []cachedSandbox{{ID: "keep"}, {ID: "drop"}}))
	require.NoError(t, setGatewayHost(ctx, "p", "gw.example.test"))
	require.NoError(t, removeSandbox(ctx, "p", "drop"))
	assert.Equal(t, "gw.example.test", getGatewayHost(ctx, "p"))
}

// newLookupAPI returns a sandbox API whose GET for testSandboxID answers with status.
func newLookupAPI(t *testing.T, status int) *sandboxAPI {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]string{
				"oidc_endpoint": "https://workspace.example.test/oidc",
			}))
			return
		}
		assert.Equal(t, sandboxPath(testSandboxID), r.URL.Path)
		w.WriteHeader(status)
		if status == http.StatusOK {
			assert.NoError(t, json.NewEncoder(w).Encode(sandboxEntry{SandboxID: testSandboxID, Status: "Running"}))
			return
		}
		_, _ = io.WriteString(w, `{"error_code": "ERROR", "message": "lookup failed"}`)
	}))
	t.Cleanup(server.Close)

	workspaceClient, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	api, err := newSandboxAPI(workspaceClient)
	require.NoError(t, err)
	return api
}

func TestResolveSandboxUsesDefault(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)
	require.NoError(t, setDefault(ctx, "profile", testSandboxID))

	got, sb, err := resolveSandbox(ctx, newLookupAPI(t, http.StatusOK), "profile", "")
	require.NoError(t, err)
	assert.Equal(t, testSandboxID, got)
	assert.Equal(t, "Running", sb.Status)
}

func TestResolveSandboxClearsMissingDefault(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)
	require.NoError(t, setDefault(ctx, "profile", testSandboxID))

	_, _, err := resolveSandbox(ctx, newLookupAPI(t, http.StatusNotFound), "profile", "")
	assert.ErrorContains(t, err, `saved default "`+testSandboxID+`" no longer exists (cleared)`)
	assert.Empty(t, getDefault(ctx, "profile"))
}

func TestResolveSandboxRejectsUnknownID(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)

	_, _, err := resolveSandbox(ctx, newLookupAPI(t, http.StatusNotFound), "profile", testSandboxID)
	assert.ErrorContains(t, err, `no sandbox named "`+testSandboxID+`"`)
}

func TestResolveSandboxFailsOnRejectedLookup(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)

	_, _, err := resolveSandbox(ctx, newLookupAPI(t, http.StatusBadRequest), "profile", testSandboxID)
	assert.ErrorContains(t, err, "failed to look up sandbox "+testSandboxID)
}

func TestResolveSandboxContinuesOnOtherLookupErrors(t *testing.T) {
	ctx, _ := stateCtx(t)
	ctx = cmdio.MockDiscard(ctx)

	got, sb, err := resolveSandbox(ctx, newLookupAPI(t, http.StatusForbidden), "profile", testSandboxID)
	require.NoError(t, err)
	assert.Equal(t, testSandboxID, got)
	assert.Nil(t, sb)
}

func TestResolveSandboxRejectsIDsThatAreNotOneSegment(t *testing.T) {
	ctx, _ := stateCtx(t)
	for _, id := range []string{".", "..", "a/../../api/2.0", `a\..\b`} {
		t.Run(id, func(t *testing.T) {
			_, _, err := resolveSandbox(ctx, nil, "profile", id)
			assert.EqualError(t, err, fmt.Sprintf("invalid sandbox ID %q", id))
		})
	}
}

func TestResolveSandboxRequiresDefault(t *testing.T) {
	ctx, _ := stateCtx(t)

	_, _, err := resolveSandbox(ctx, nil, "profile", "")
	assert.ErrorContains(t, err, "no default sandbox configured")
}
