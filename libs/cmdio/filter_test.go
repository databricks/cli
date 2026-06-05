package cmdio

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stableURL struct {
	Name               string `json:"name"`
	InitialWorkspaceID string `json:"initial_workspace_id"`
	URL                string `json:"url,omitempty"`
}

type stableURLList struct {
	StableURLs []stableURL `json:"stable_urls"`
}

type wrapper struct {
	StableURL stableURL `json:"stable_url"`
}

func TestApplyInputOnlyMaskEmptyPathsReturnsValueUnchanged(t *testing.T) {
	in := stableURL{Name: "n", InitialWorkspaceID: "w"}
	out, err := applyInputOnlyMask(in, nil)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestApplyInputOnlyMaskFlatField(t *testing.T) {
	in := stableURL{Name: "n", InitialWorkspaceID: "w", URL: "u"}
	out, err := applyInputOnlyMask(in, []string{"initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"n","url":"u"}`, string(b))
}

func TestApplyInputOnlyMaskFieldAbsentIsNoop(t *testing.T) {
	in := stableURL{Name: "n"}
	out, err := applyInputOnlyMask(in, []string{"missing_field"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	// Name retained; missing path silently ignored. InitialWorkspaceID
	// stays present at "" because the struct tag has no omitempty.
	assert.JSONEq(t, `{"name":"n","initial_workspace_id":""}`, string(b))
}

func TestApplyInputOnlyMaskNested(t *testing.T) {
	in := wrapper{StableURL: stableURL{Name: "n", InitialWorkspaceID: "w"}}
	out, err := applyInputOnlyMask(in, []string{"stable_url.initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"stable_url":{"name":"n"}}`, string(b))
}

func TestApplyInputOnlyMaskSliceElements(t *testing.T) {
	in := stableURLList{StableURLs: []stableURL{
		{Name: "a", InitialWorkspaceID: "1"},
		{Name: "b", InitialWorkspaceID: "2"},
	}}
	out, err := applyInputOnlyMask(in, []string{"stable_urls.initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"stable_urls":[{"name":"a"},{"name":"b"}]}`, string(b))
}

func TestApplyInputOnlyMaskMultiplePaths(t *testing.T) {
	in := stableURL{Name: "n", InitialWorkspaceID: "w", URL: "u"}
	out, err := applyInputOnlyMask(in, []string{"initial_workspace_id", "url"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"n"}`, string(b))
}

// TestRenderFilteredStripsInputOnlyField is the integration check: pass
// the same StableUrl-like value the CLI would receive from the SDK, and
// confirm the rendered JSON does not contain initial_workspace_id.
func TestRenderFilteredStripsInputOnlyField(t *testing.T) {
	v := stableURL{Name: "accounts/x/stable-urls/y", InitialWorkspaceID: "ws-1", URL: "https://example.test"}

	out := &bytes.Buffer{}
	c := &cmdIO{
		capabilities: Capabilities{},
		outputFormat: flags.OutputJSON,
		out:          out,
		err:          out,
	}
	ctx := InContext(t.Context(), c)
	require.NoError(t, RenderFiltered(ctx, v, []string{"initial_workspace_id"}))

	assert.JSONEq(t, `{"name":"accounts/x/stable-urls/y","url":"https://example.test"}`, out.String())
	assert.NotContains(t, out.String(), "initial_workspace_id")
}

func TestRenderFilteredNoPathsMatchesRender(t *testing.T) {
	v := stableURL{Name: "n", InitialWorkspaceID: "w"}

	want := &bytes.Buffer{}
	got := &bytes.Buffer{}
	mk := func(buf *bytes.Buffer) *cmdIO {
		return &cmdIO{capabilities: Capabilities{}, outputFormat: flags.OutputJSON, out: buf, err: buf}
	}
	require.NoError(t, Render(InContext(t.Context(), mk(want)), v))
	require.NoError(t, RenderFiltered(InContext(t.Context(), mk(got)), v, nil))
	assert.Equal(t, want.String(), got.String())
}
