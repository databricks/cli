package direct

import (
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectDependencyCycle(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		cycle string
	}{
		{
			name: "chain",
			yaml: `
resources:
  jobs:
    a: {name: "${resources.jobs.b.id}"}
    b: {name: "b"}
`,
		},
		{
			// The permissions are a separate node that depends on the job.
			name: "permissions_refer_to_own_job",
			yaml: `
resources:
  jobs:
    a:
      name: a
      permissions:
        - level: CAN_VIEW
          user_name: "${resources.jobs.a.name}"
`,
		},
		{
			// References to anything but resources are not edges.
			name: "non_resource_reference",
			yaml: `
resources:
  jobs:
    a: {name: "${artifacts.x.path} ${workspace.file_path}"}
`,
		},
		{
			// Empty grants are not planned, so depending on them is no edge.
			name: "depends_on_empty_grants",
			yaml: `
resources:
  schemas:
    s:
      catalog_name: main
      name: s
      grants: []
      lifecycle:
        depends_on:
          - resource: schemas.s.grants
`,
		},
		{
			// makePlan cannot read a resource whose key has a dot; the check leaves it to the plan.
			name: "dotted_resource_key",
			yaml: `
resources:
  jobs:
    "foo.bar": {name: "x"}
`,
		},
		{
			// A reference the plan resolves later does not hide a cycle elsewhere.
			name: "cycle_next_to_non_resource_reference",
			yaml: `
resources:
  jobs:
    a: {name: "${artifacts.x.path}"}
    b: {name: "${resources.jobs.b.id}"}
`,
			cycle: "cycle detected: resources.jobs.b refers to itself via ${resources.jobs.b.id}",
		},
		{
			name: "self",
			yaml: `
resources:
  jobs:
    a: {name: "${resources.jobs.a.id}"}
`,
			cycle: "cycle detected: resources.jobs.a refers to itself via ${resources.jobs.a.id}",
		},
		{
			name: "two_jobs",
			yaml: `
resources:
  jobs:
    a: {name: "${resources.jobs.b.id}"}
    b: {name: "${resources.jobs.a.url}"}
`,
			cycle: "cycle detected: resources.jobs.b refers to resources.jobs.a via ${resources.jobs.a.url} which refers to resources.jobs.b via ${resources.jobs.b.id}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tc.yaml))
			require.Empty(t, diags)

			cycle := DetectDependencyCycle(t.Context(), root, nil)
			if tc.cycle == "" {
				assert.Nil(t, cycle)
				return
			}
			require.NotNil(t, cycle)
			assert.Equal(t, tc.cycle, cycle.Error())
		})
	}
}

func TestDetectDependencyCycleLeavesStateUnchanged(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    a: {name: "a"}
`))
	require.Empty(t, diags)

	// A job removed from the config is planned for deletion with the dependencies recorded in
	// state; one of them points at a resource that is in neither the config nor the state.
	db := &dstate.Database{State: map[string]dstate.ResourceEntry{
		"resources.jobs.gone": {
			ID: "1",
			DependsOn: []deployplan.DependsOnEntry{
				{Node: "resources.jobs.missing", Label: "${resources.jobs.missing.id}"},
				{Node: "resources.jobs.a", Label: "${resources.jobs.a.id}"},
			},
		},
	}}
	want := slices.Clone(db.State["resources.jobs.gone"].DependsOn)

	assert.Nil(t, DetectDependencyCycle(t.Context(), root, db))
	assert.Equal(t, want, db.State["resources.jobs.gone"].DependsOn)
}
