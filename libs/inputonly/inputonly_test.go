package inputonly

import (
	"encoding/json"
	"testing"

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

type taggedStableURLs struct {
	Tags map[string]stableURL `json:"tags"`
}

func TestStripEmptyPathsReturnsValueUnchanged(t *testing.T) {
	in := stableURL{Name: "n", InitialWorkspaceID: "w"}
	out, err := Strip(in, nil)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestStripFlatField(t *testing.T) {
	in := stableURL{Name: "n", InitialWorkspaceID: "w", URL: "u"}
	out, err := Strip(in, []string{"initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"n","url":"u"}`, string(b))
}

func TestStripFieldAbsentIsNoop(t *testing.T) {
	in := stableURL{Name: "n"}
	out, err := Strip(in, []string{"missing_field"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	// Name retained; missing path silently ignored. InitialWorkspaceID stays
	// present at "" because the struct tag has no omitempty.
	assert.JSONEq(t, `{"name":"n","initial_workspace_id":""}`, string(b))
}

func TestStripNested(t *testing.T) {
	in := wrapper{StableURL: stableURL{Name: "n", InitialWorkspaceID: "w"}}
	out, err := Strip(in, []string{"stable_url.initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"stable_url":{"name":"n"}}`, string(b))
}

func TestStripSliceElements(t *testing.T) {
	in := stableURLList{StableURLs: []stableURL{
		{Name: "a", InitialWorkspaceID: "1"},
		{Name: "b", InitialWorkspaceID: "2"},
	}}
	out, err := Strip(in, []string{"stable_urls.initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"stable_urls":[{"name":"a"},{"name":"b"}]}`, string(b))
}

func TestStripMultiplePaths(t *testing.T) {
	in := stableURL{Name: "n", InitialWorkspaceID: "w", URL: "u"}
	out, err := Strip(in, []string{"initial_workspace_id", "url"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"n"}`, string(b))
}

// TestStripPreservesLargeInt64 guards against the float64 round-trip pitfall:
// encoding/json decodes JSON numbers into float64 (53-bit mantissa) when the
// destination is `any`, which silently loses precision for SDK fields like
// spark_context_id (int64) above 2^53. Strip uses json.Number to side-step
// that.
func TestStripPreservesLargeInt64(t *testing.T) {
	type clusterResponse struct {
		ClusterName        string `json:"cluster_name"`
		SparkContextID     int64  `json:"spark_context_id"`
		InitialWorkspaceID string `json:"initial_workspace_id"`
	}
	in := clusterResponse{
		ClusterName:        "c",
		SparkContextID:     9007199254740993, // 2^53 + 1, unrepresentable as float64
		InitialWorkspaceID: "ws-1",
	}
	out, err := Strip(in, []string{"initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"spark_context_id":9007199254740993`)
	assert.NotContains(t, string(b), "initial_workspace_id")
}

// TestStripDoesNotMatchAnywhere guards against the implicit-fallback pitfall:
// an anchored path that doesn't match at the level it targets must NOT
// silently descend into other objects and strip same-named fields elsewhere.
// Since INPUT_ONLY fields are always omitted by the server, the literal miss
// at the anchored level is the common case — a fallback descent would fire
// every time and quietly strip legitimate fields with the same leaf name.
func TestStripDoesNotMatchAnywhere(t *testing.T) {
	type details struct {
		Name string `json:"name"`
		Size string `json:"size"`
	}
	type response struct {
		ID      string  `json:"id"`
		Details details `json:"details"`
	}
	in := response{ID: "123", Details: details{Name: "keep-me", Size: "L"}}
	out, err := Strip(in, []string{"name"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"123","details":{"name":"keep-me","size":"L"}}`, string(b))
}

// TestStripDoesNotDescendIntoMaps documents the strict-map behavior: a path
// that doesn't literally match a map key is a no-op, never a fallback into
// every value. cligen does not emit paths that descend through proto maps
// today (cli.json carries no map value refs); the runtime matches that
// contract instead of guessing.
func TestStripDoesNotDescendIntoMaps(t *testing.T) {
	in := taggedStableURLs{Tags: map[string]stableURL{
		"env":  {Name: "a", InitialWorkspaceID: "1"},
		"prod": {Name: "b", InitialWorkspaceID: "2"},
	}}
	out, err := Strip(in, []string{"tags.initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)

	var got struct {
		Tags map[string]map[string]any `json:"tags"`
	}
	require.NoError(t, json.Unmarshal(b, &got))
	require.Len(t, got.Tags, 2)
	assert.Equal(t, "1", got.Tags["env"]["initial_workspace_id"])
	assert.Equal(t, "2", got.Tags["prod"]["initial_workspace_id"])
}
