package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureGrantsDependencies(t *testing.T) {
	grants := func(principal string) []catalog.PrivilegeAssignment {
		return []catalog.PrivilegeAssignment{{Principal: principal, Privileges: []catalog.Privilege{catalog.PrivilegeCreateTable}}}
	}
	schema := func(grants []catalog.PrivilegeAssignment) *resources.Schema {
		return &resources.Schema{CatalogName: "main", Name: "src", Grants: grants}
	}
	// A pipeline that refers to the schema.
	pipeline := &resources.Pipeline{
		Catalog: "${resources.schemas.src.catalog_name}",
		Schema:  "${resources.schemas.src.name}",
	}
	// An app that refers to the schema.
	app := &resources.App{Name: "app", Description: "${resources.schemas.src.name}"}

	tests := []struct {
		name      string
		resources config.Resources
		expected  map[string][]resources.DependsOn
	}{
		{
			name: "resource that refers to a resource with grants depends on the grants",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema(grants("sp"))},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "resource without grants",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema(nil)},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline},
			},
			expected: map[string][]resources.DependsOn{},
		},
		{
			name: "each resource depends on the grants of the resource it refers to",
			resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}, Grants: grants("sp")}},
				Schemas: map[string]*resources.Schema{"src": {
					CreateSchema: catalog.CreateSchema{CatalogName: "${resources.catalogs.main.name}", Name: "src"},
					Grants:       grants("sp"),
				}},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline},
			},
			expected: map[string][]resources.DependsOn{
				"schemas.src":    {{Resource: "catalogs.main.grants"}},
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "grants that depend on the referring resource",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema(grants("${resources.apps.app.service_principal_client_id}"))},
				Apps:      map[string]*resources.App{"app": app},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "grants that transitively depend on the referring resource",
			resources: config.Resources{
				Schemas: map[string]*resources.Schema{"src": schema(grants("${resources.jobs.job.run_as.service_principal_name}"))},
				Jobs:    map[string]*resources.Job{"job": {JobSettings: jobs.JobSettings{Name: "${resources.apps.app.name}"}}},
				Apps:    map[string]*resources.App{"app": app},
			},
			expected: map[string][]resources.DependsOn{},
		},
		{
			name: "lifecycle.depends_on on a resource implies its grants",
			resources: config.Resources{
				Schemas: map[string]*resources.Schema{"src": schema(grants("sp"))},
				Jobs: map[string]*resources.Job{"job": {
					BaseResource: resources.BaseResource{Lifecycle: resources.Lifecycle{DependsOn: []resources.DependsOn{{Resource: "schemas.src"}}}},
					JobSettings:  jobs.JobSettings{Name: "job"},
				}},
			},
			expected: map[string][]resources.DependsOn{
				"jobs.job": {{Resource: "schemas.src"}, {Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "grants already in lifecycle.depends_on",
			resources: config.Resources{
				Schemas: map[string]*resources.Schema{"src": schema(grants("sp"))},
				Pipelines: map[string]*resources.Pipeline{"load": {
					BaseResource:   resources.BaseResource{Lifecycle: resources.Lifecycle{DependsOn: []resources.DependsOn{{Resource: "schemas.src.grants"}}}},
					CreatePipeline: pipeline.CreatePipeline,
				}},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{Config: config.Root{Resources: tc.resources}}

			// Normalization runs more than once, so applying the mutator again must not add anything.
			for range 2 {
				diags := bundle.Apply(t.Context(), b, CaptureGrantsDependencies())
				require.NoError(t, diags.Error())

				actual := make(map[string][]resources.DependsOn)
				for _, group := range b.Config.Resources.AllResources() {
					for key, resource := range group.Resources {
						if deps := resource.GetLifecycle().GetDependsOn(); len(deps) > 0 {
							actual[group.Description.PluralName+"."+key] = deps
						}
					}
				}
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}
