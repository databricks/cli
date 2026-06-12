package main

import (
	"testing"

	"github.com/databricks/cli/internal/clijson"
	"github.com/stretchr/testify/assert"
)

func TestInputOnlyPathsFlat(t *testing.T) {
	schemas := map[string]*clijson.SchemaJSON{
		"dr.StableUrl": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"name":                 {},
				"initial_workspace_id": {Behaviors: []string{"IMMUTABLE", "INPUT_ONLY"}},
				"url":                  {Behaviors: []string{"OUTPUT_ONLY"}},
			},
		},
	}
	assert.Equal(t, []string{"initial_workspace_id"}, inputOnlyPaths(schemas, "dr.StableUrl"))
}

func TestInputOnlyPathsNestedMessage(t *testing.T) {
	// A singleton message ref is followed; an INPUT_ONLY field on the
	// referenced type emits a dotted path.
	schemas := map[string]*clijson.SchemaJSON{
		"x.Wrapper": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"inner": {Ref: "x.Inner"},
			},
		},
		"x.Inner": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"secret": {Behaviors: []string{"INPUT_ONLY"}},
				"name":   {},
			},
		},
	}
	assert.Equal(t, []string{"inner.secret"}, inputOnlyPaths(schemas, "x.Wrapper"))
}

func TestInputOnlyPathsInputOnlyMessageNotRecursedInto(t *testing.T) {
	// A field whose type is itself INPUT_ONLY emits a single path; the
	// whole subtree is stripped at runtime so its inner fields don't need
	// their own paths.
	schemas := map[string]*clijson.SchemaJSON{
		"x.Outer": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"creds": {Ref: "x.Creds", Behaviors: []string{"INPUT_ONLY"}},
			},
		},
		"x.Creds": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"password": {Behaviors: []string{"INPUT_ONLY"}},
			},
		},
	}
	assert.Equal(t, []string{"creds"}, inputOnlyPaths(schemas, "x.Outer"))
}

func TestInputOnlyPathsCycle(t *testing.T) {
	// A tree-shaped resource that references itself: the walker doesn't
	// loop and still picks up the INPUT_ONLY field at this level.
	schemas := map[string]*clijson.SchemaJSON{
		"x.Tree": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"secret": {Behaviors: []string{"INPUT_ONLY"}},
				"parent": {Ref: "x.Tree"},
			},
		},
	}
	assert.Equal(t, []string{"secret"}, inputOnlyPaths(schemas, "x.Tree"))
}

func TestInputOnlyPathsUnknownRoot(t *testing.T) {
	assert.Nil(t, inputOnlyPaths(map[string]*clijson.SchemaJSON{}, "x.Missing"))
}

func TestEligibleForInputOnly(t *testing.T) {
	cases := []struct {
		name   string
		method *MethodJSON
		want   bool
	}{
		{
			name:   "standard sync method",
			method: &MethodJSON{Response: &EntityJSON{PascalName: "StableUrl"}},
			want:   true,
		},
		{
			name:   "empty response",
			method: &MethodJSON{Response: &EntityJSON{PascalName: "Empty", IsEmptyResponse: true}},
			want:   false,
		},
		{
			name:   "no response",
			method: &MethodJSON{},
			want:   false,
		},
		{
			name:   "paginated",
			method: &MethodJSON{Response: &EntityJSON{PascalName: "ListResponse"}, Pagination: &PaginationJSON{}},
			want:   false,
		},
		{
			name:   "byte stream",
			method: &MethodJSON{Response: &EntityJSON{PascalName: "Body"}, IsResponseByteStream: true},
			want:   false,
		},
		{
			name:   "long-running",
			method: &MethodJSON{Response: &EntityJSON{PascalName: "Operation"}, LongRunningOperation: &LROJSON{}},
			want:   false,
		},
		{
			name:   "wait",
			method: &MethodJSON{Response: &EntityJSON{PascalName: "Resp"}, Wait: &WaitJSON{}},
			want:   false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, eligibleForInputOnly(c.method))
		})
	}
}
