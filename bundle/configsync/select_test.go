package configsync

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterChanges(t *testing.T) {
	changes := Changes{
		"resources.jobs.foo": {
			"max_concurrent_runs": {Operation: OperationReplace, Value: 5},
		},
		"resources.jobs.foo.permissions": {
			"[0].level": {Operation: OperationReplace, Value: "CAN_MANAGE"},
		},
		// Boundary: shares the "resources.jobs.foo" prefix but is a different
		// resource, so selecting "resources.jobs.foo" must not pull it in.
		"resources.jobs.foobar": {
			"name": {Operation: OperationReplace, Value: "foobar"},
		},
		"resources.jobs.bar": {
			"name": {Operation: OperationReplace, Value: "bar"},
		},
		"resources.schemas.baz": {
			"comment": {Operation: OperationAdd, Value: "c"},
		},
		"resources.schemas.baz.grants": {
			"[0].principal": {Operation: OperationAdd, Value: "users"},
		},
		// A resource whose name is literally "permissions" is the resource itself,
		// not a sub-node, and is kept only when selected by its own key.
		"resources.jobs.permissions": {
			"name": {Operation: OperationReplace, Value: "p"},
		},
	}

	tests := []struct {
		name     string
		selected []string
		wantKeys []string
	}{
		{
			name:     "resource groups its permissions sub-node by prefix, excludes the foobar sibling",
			selected: []string{"resources.jobs.foo"},
			wantKeys: []string{"resources.jobs.foo", "resources.jobs.foo.permissions"},
		},
		{
			name:     "resource without sub-nodes",
			selected: []string{"resources.jobs.bar"},
			wantKeys: []string{"resources.jobs.bar"},
		},
		{
			name:     "grants sub-node follows its parent",
			selected: []string{"resources.schemas.baz"},
			wantKeys: []string{"resources.schemas.baz", "resources.schemas.baz.grants"},
		},
		{
			name:     "multiple selections are a union",
			selected: []string{"resources.jobs.bar", "resources.schemas.baz"},
			wantKeys: []string{"resources.jobs.bar", "resources.schemas.baz", "resources.schemas.baz.grants"},
		},
		{
			name:     "resource with no detected changes yields empty result",
			selected: []string{"resources.jobs.never_drifted"},
			wantKeys: []string{},
		},
		{
			name:     "resource named permissions is kept only by its own key",
			selected: []string{"resources.jobs.permissions"},
			wantKeys: []string{"resources.jobs.permissions"},
		},
		{
			name:     "empty selection keeps nothing",
			selected: nil,
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterChanges(changes, tt.selected)
			assert.ElementsMatch(t, tt.wantKeys, slices.Collect(maps.Keys(got)))
			for _, key := range tt.wantKeys {
				assert.Equal(t, changes[key], got[key])
			}
		})
	}

	// The input map is never mutated.
	assert.Len(t, changes, 7)
}
