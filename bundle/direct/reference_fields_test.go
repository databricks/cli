package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindUnknownReferenceFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{
			name: "unknown_field",
			yaml: `
resources:
  volumes:
    bar: {catalog_name: c, schema_name: s, name: bar}
    foo: {catalog_name: "${resources.volumes.bar.non_existent}", schema_name: s, name: foo}
`,
			want: []string{`resources.volumes.foo.catalog_name: resources.volumes.bar.non_existent: resource type "volumes" has no field "non_existent"`},
		},
		{
			name: "unknown_nested_field",
			yaml: `
resources:
  jobs:
    a: {name: a, tasks: [{task_key: t}]}
    b: {name: "x ${resources.jobs.a.tasks[0].nope} ${resources.jobs.a.tasks[0].nope}"}
`,
			want: []string{`resources.jobs.b.name: resources.jobs.a.tasks[0].nope: resource type "jobs" has no field "tasks[0].nope"`},
		},
		{
			// Fields in the state or only in the remote type, the id, map keys, Terraform-style
			// paths and permissions and grants entries all resolve.
			name: "known_fields",
			yaml: `
resources:
  apps:
    myapp: {name: myapp, source_code_path: ./app}
  pipelines:
    p: {name: p}
  schemas:
    s:
      catalog_name: main
      name: s
      grants: [{principal: x, privileges: [USE_SCHEMA]}]
  jobs:
    a:
      name: a
      tags: {k: v}
      permissions: [{level: CAN_VIEW, user_name: u}]
    b:
      name: "${resources.jobs.a.name} ${resources.jobs.a.id} ${resources.jobs.a.tags.k} ${resources.jobs.a.task[0].task_key}"
      description: "${resources.apps.myapp.url} ${resources.pipelines.p.state} ${resources.pipelines.p.creator_user_name}"
      tags:
        x: "${resources.jobs.a.permissions[0].level} ${resources.schemas.s.grants[0].principal} ${resources.jobs.a}"
`,
		},
		{
			// Left to TFOnlyReferences.
			name: "terraform_only_field",
			yaml: `
resources:
  jobs:
    a: {name: a}
    b: {name: "${resources.jobs.a.always_running}"}
`,
		},
		{
			// The plan drops input-only fields such as volume_path, with the references in them.
			name: "input_only_field",
			yaml: `
resources:
  volumes:
    bar: {catalog_name: c, schema_name: s, name: bar}
    foo: {catalog_name: c, schema_name: s, name: foo, volume_path: "${resources.volumes.bar.non_existent}"}
`,
		},
		{
			name: "not_resource_references",
			yaml: `
resources:
  jobs:
    a: {name: "${var.x} ${workspace.file_path} ${resources.unknown_type.a.b}"}
`,
		},
		{
			// The plan cannot read a resource whose key has a dot; it reports that itself.
			name: "dotted_resource_key",
			yaml: `
resources:
  jobs:
    a: {name: a}
    "foo.bar": {name: "${resources.jobs.a.non_existent}"}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tc.yaml))
			require.Empty(t, diags)

			found, err := FindUnknownReferenceFields(root.Value())
			require.NoError(t, err)
			var got []string
			for _, f := range found {
				got = append(got, f.Path.String()+": "+f.Reference+": "+f.Err.Error())
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
