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

type nestedMapWrapper struct {
	Spec struct {
		Tags map[string]stableURL `json:"tags"`
	} `json:"spec"`
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

func TestStripMapValues(t *testing.T) {
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
	for k, v := range got.Tags {
		assert.NotContains(t, v, "initial_workspace_id", "tag %q should have initial_workspace_id stripped", k)
		assert.Contains(t, v, "name", "tag %q should retain name", k)
	}
}

func TestStripNestedInMapValue(t *testing.T) {
	// Path lands inside a map at the second segment, then descends two more
	// levels into the map's value. Confirms map transparency composes with
	// regular literal-key descent.
	var in nestedMapWrapper
	in.Spec.Tags = map[string]stableURL{
		"a": {Name: "x", InitialWorkspaceID: "1"},
		"b": {Name: "y", InitialWorkspaceID: "2"},
	}
	out, err := Strip(in, []string{"spec.tags.initial_workspace_id"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)

	var got struct {
		Spec struct {
			Tags map[string]map[string]any `json:"tags"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(b, &got))
	require.Len(t, got.Spec.Tags, 2)
	for k, v := range got.Spec.Tags {
		assert.NotContains(t, v, "initial_workspace_id", "spec.tags[%q] should have initial_workspace_id stripped", k)
	}
}

func TestStripMultiplePaths(t *testing.T) {
	in := stableURL{Name: "n", InitialWorkspaceID: "w", URL: "u"}
	out, err := Strip(in, []string{"initial_workspace_id", "url"})
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"n"}`, string(b))
}
