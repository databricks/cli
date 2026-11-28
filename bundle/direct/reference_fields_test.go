package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
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
			// Reported once per value, with the field as written (Terraform-style "task").
			name: "unknown_nested_field",
			yaml: `
resources:
  jobs:
    a: {name: a, tasks: [{task_key: t}]}
    b: {name: "x ${resources.jobs.a.task[0].nope} ${resources.jobs.a.task[0].nope}"}
`,
			want: []string{`resources.jobs.b.name: resources.jobs.a.task[0].nope: resource type "jobs" has no field "task[0].nope"`},
		},
		{
			// Permissions and grants are nodes of their own; the error names the field relative to
			// the resource.
			name: "unknown_permissions_and_grants_fields",
			yaml: `
resources:
  schemas:
    s:
      catalog_name: main
      name: s
      grants: [{principal: x, privileges: [USE_SCHEMA]}]
  jobs:
    a:
      name: a
      permissions: [{level: CAN_VIEW, user_name: u}]
    b: {name: "${resources.jobs.a.permissions[0].nope} ${resources.schemas.s.grants[0].nope}"}
`,
			want: []string{
				`resources.jobs.b.name: resources.jobs.a.permissions[0].nope: resource type "jobs" has no field "permissions[0].nope"`,
				`resources.jobs.b.name: resources.schemas.s.grants[0].nope: resource type "schemas" has no field "grants[0].nope"`,
			},
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
		{
			// The plan reports a missing resource as "no such node".
			name: "missing_target_resource",
			yaml: `
resources:
  jobs:
    a: {name: "${resources.jobs.absent.non_existent}"}
`,
		},
		{
			// A reference standing for all grants becomes empty grants, which a first deploy
			// does not plan, so it never resolves the reference.
			name: "reference_in_empty_new_node",
			yaml: `
resources:
  schemas:
    s:
      catalog_name: main
      name: s
      grants: "${resources.schemas.s.non_existent}"
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tc.yaml))
			require.Empty(t, diags)

			found, err := FindUnknownReferenceFields(root, nil)
			require.NoError(t, err)
			var got []string
			for _, f := range found {
				got = append(got, f.Path.String()+": "+f.Reference+": "+f.Err.Error())
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

// With state, the plan keeps an empty node and resolves its references.
func TestFindUnknownReferenceFieldsEmptyNodeWithState(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  schemas:
    s:
      catalog_name: main
      name: s
      grants: "${resources.schemas.s.non_existent}"
`))
	require.Empty(t, diags)
	db := &dstate.Database{State: map[string]dstate.ResourceEntry{"resources.schemas.s.grants": {ID: "main.s"}}}

	found, err := FindUnknownReferenceFields(root, db)
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "resources.schemas.s.grants", found[0].Path.String())
	assert.EqualError(t, found[0].Err, `resource type "schemas" has no field "non_existent"`)
}

// plannedNodes must select the nodes makePlan plans from the config, or the check and the plan
// disagree on which references to resolve.
func TestPlannedNodesMatchMakePlan(t *testing.T) {
	yaml := `
resources:
  schemas:
    full:
      catalog_name: main
      name: full
      grants: [{principal: x, privileges: [USE_SCHEMA]}]
    empty_grants:
      catalog_name: main
      name: empty_grants
      grants: []
    empty_grants_in_state:
      catalog_name: main
      name: empty_grants_in_state
      grants: []
  jobs:
    a:
      name: a
      permissions: [{level: CAN_VIEW, user_name: u}]
    b:
      name: b
      permissions: []
`
	for _, db := range []*dstate.Database{
		{},
		{State: map[string]dstate.ResourceEntry{
			"resources.schemas.empty_grants_in_state.grants": {ID: "main.empty_grants_in_state"},
			"resources.jobs.b.permissions":                   {ID: "/jobs/1"},
		}},
	} {
		root, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
		require.Empty(t, diags)
		adapters, err := dresources.InitAll(nil)
		require.NoError(t, err)

		b := DeploymentBundle{Adapters: adapters}
		plan, err := b.makePlan(t.Context(), root, db)
		require.NoError(t, err)
		want := make(map[string]bool)
		for node, entry := range plan.Plan {
			if entry.Action != deployplan.Delete {
				want[node] = true
			}
		}

		assert.Equal(t, want, plannedNodes(root, db, adapters))
	}
}
