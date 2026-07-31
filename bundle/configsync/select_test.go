package configsync

import (
	"fmt"
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

func TestDescribeStateIDs(t *testing.T) {
	tests := []struct {
		name         string
		byTypeID     map[string]string
		resourceType string
		want         string
	}{
		{
			name:         "empty state is called out explicitly",
			byTypeID:     map[string]string{},
			resourceType: "jobs",
			want:         "the deployment state contains no resources with ids (the bundle may not be deployed, or its resource state is missing)",
		},
		{
			// Permissions/grants are indexed under the parent's type with a
			// path-shaped object id; reporting them would leak a path and
			// inflate the counts.
			name: "permissions and grants sub-resources are excluded",
			byTypeID: map[string]string{
				"jobs:111":            "resources.jobs.a",
				"jobs:/jobs/111":      "resources.jobs.a.permissions",
				"schemas:s1":          "resources.schemas.s",
				"schemas:/schemas/s1": "resources.schemas.s.grants",
			},
			resourceType: "jobs",
			want:         "deployed jobs ids in state: 111",
		},
		{
			name: "state holding only sub-resources reads as no resources with ids",
			byTypeID: map[string]string{
				"jobs:/jobs/111": "resources.jobs.a.permissions",
			},
			resourceType: "jobs",
			want:         "the deployment state contains no resources with ids (the bundle may not be deployed, or its resource state is missing)",
		},
		{
			// A resource literally named "permissions" is a real resource, not a
			// sub-resource, so its id must still be reported.
			name: "resource named permissions is still reported",
			byTypeID: map[string]string{
				"jobs:333": "resources.jobs.permissions",
			},
			resourceType: "jobs",
			want:         "deployed jobs ids in state: 333",
		},
		{
			name: "state holds other types only",
			byTypeID: map[string]string{
				"pipelines:abc-123": "resources.pipelines.p",
				"schemas:99":        "resources.schemas.s",
				"pipelines:def-456": "resources.pipelines.q",
			},
			resourceType: "jobs",
			want:         "the deployment state contains no jobs resources (deployed resources by type: pipelines=2, schemas=1)",
		},
		{
			name: "ids of the requested type are listed sorted",
			byTypeID: map[string]string{
				"jobs:222":  "resources.jobs.b",
				"jobs:111":  "resources.jobs.a",
				"schemas:7": "resources.schemas.s",
			},
			resourceType: "jobs",
			want:         "deployed jobs ids in state: 111, 222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, describeStateIDs(tt.byTypeID, tt.resourceType))
		})
	}
}

func TestDescribeStateIDsTruncatesLongLists(t *testing.T) {
	byTypeID := make(map[string]string)
	for i := range maxReportedIDs + 3 {
		// Ids are zero-padded so lexical sort order is also numeric order,
		// keeping the assertion independent of id formatting.
		id := fmt.Sprintf("%03d", i)
		byTypeID["jobs:"+id] = "resources.jobs.j" + id
	}

	got := describeStateIDs(byTypeID, "jobs")
	assert.Equal(t, "deployed jobs ids in state: 000, 001, 002, 003, 004, 005, 006, 007, 008, 009 (and 3 more)", got)
}
