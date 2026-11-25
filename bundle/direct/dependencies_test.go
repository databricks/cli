package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
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

			cycle := DetectDependencyCycle(t.Context(), root)
			if tc.cycle == "" {
				assert.Nil(t, cycle)
				return
			}
			require.NotNil(t, cycle)
			assert.Equal(t, tc.cycle, cycle.Error())
		})
	}
}
