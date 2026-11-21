package resourcemutator

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapturePipelineDependencies(t *testing.T) {
	grants := func(principal string) []catalog.PrivilegeAssignment {
		return []catalog.PrivilegeAssignment{{Principal: principal, Privileges: []catalog.Privilege{catalog.PrivilegeCreateTable}}}
	}
	schema := func(grants []catalog.PrivilegeAssignment) *resources.Schema {
		return &resources.Schema{CatalogName: "main", Name: "src", Grants: grants}
	}
	// A pipeline that refers to the schema. A constructor, because the mutator writes to the
	// pipeline and a shared pointer would leak lifecycle.depends_on between cases.
	pipeline := func() *resources.Pipeline {
		return &resources.Pipeline{
			Catalog: "${resources.schemas.src.catalog_name}",
			Schema:  "${resources.schemas.src.name}",
		}
	}
	// A job with permissions, referred to by name.
	job := func(permissions []resources.JobPermission) *resources.Job {
		return &resources.Job{
			Name:        "job",
			Permissions: permissions,
		}
	}

	tests := []struct {
		name      string
		resources config.Resources
		expected  map[string][]resources.DependsOn
	}{
		{
			name: "pipeline that refers to a resource with grants depends on the grants",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema(grants("sp"))},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline()},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "empty grants and permissions add nothing",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema([]catalog.PrivilegeAssignment{})},
				Jobs:      map[string]*resources.Job{"job": job([]resources.JobPermission{})},
				Pipelines: map[string]*resources.Pipeline{"load": {CreatePipeline: pipelines.CreatePipeline{Name: "${resources.jobs.job.name}", Catalog: "${resources.schemas.src.catalog_name}", Schema: "${resources.schemas.src.name}"}}},
			},
			expected: map[string][]resources.DependsOn{},
		},
		{
			name: "null pipeline entry does not panic and other pipelines are still processed",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema(grants("sp"))},
				Pipelines: map[string]*resources.Pipeline{"empty": nil, "load": pipeline()},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "pipeline waits for the grants of the schema and of its catalog",
			resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}, Grants: grants("sp")}},
				Schemas: map[string]*resources.Schema{"src": {
					CreateSchema: catalog.CreateSchema{CatalogName: "${resources.catalogs.main.name}", Name: "src"},
					Grants:       grants("sp"),
				}},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline()},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}, {Resource: "catalogs.main.grants"}},
			},
		},
		{
			name: "pipeline that refers to a volume waits for the grants of the volume, its schema and its catalog",
			resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}, Grants: grants("sp")}},
				Schemas: map[string]*resources.Schema{"src": {
					CreateSchema: catalog.CreateSchema{CatalogName: "${resources.catalogs.main.name}", Name: "src"},
					Grants:       grants("sp"),
				}},
				Volumes: map[string]*resources.Volume{"raw": {
					CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
						CatalogName: "${resources.catalogs.main.name}",
						SchemaName:  "${resources.schemas.src.name}",
						Name:        "raw",
					},
					Grants: grants("sp"),
				}},
				Pipelines: map[string]*resources.Pipeline{"load": {
					CreatePipeline: pipelines.CreatePipeline{Catalog: "${resources.volumes.raw.catalog_name}", Schema: "${resources.volumes.raw.schema_name}"},
				}},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "volumes.raw.grants"}, {Resource: "catalogs.main.grants"}, {Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "non-pipeline resource that refers to a resource with grants gets nothing",
			resources: config.Resources{
				Schemas: map[string]*resources.Schema{"src": schema(grants("sp"))},
				Jobs:    map[string]*resources.Job{"job": {JobSettings: jobs.JobSettings{Name: "${resources.schemas.src.name}"}}},
			},
			expected: map[string][]resources.DependsOn{},
		},
		{
			name: "grants that refer to an app that does not refer to the pipeline are added",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema(grants("${resources.apps.app.service_principal_client_id}"))},
				Apps:      map[string]*resources.App{"app": {Name: "app", Description: "${resources.schemas.src.name}"}},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline()},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "grants that depend on an app that refers to the pipeline are skipped",
			resources: config.Resources{
				Schemas:   map[string]*resources.Schema{"src": schema(grants("${resources.apps.app.service_principal_client_id}"))},
				Apps:      map[string]*resources.App{"app": {Name: "app", Description: "${resources.pipelines.load.name}"}},
				Pipelines: map[string]*resources.Pipeline{"load": pipeline()},
			},
			expected: map[string][]resources.DependsOn{},
		},
		{
			name: "lifecycle.depends_on on a pipeline implies the grants of the resource",
			resources: config.Resources{
				Schemas: map[string]*resources.Schema{"src": schema(grants("sp"))},
				Pipelines: map[string]*resources.Pipeline{"load": {
					BaseResource:   resources.BaseResource{Lifecycle: resources.Lifecycle{DependsOn: []resources.DependsOn{{Resource: "schemas.src"}}}},
					CreatePipeline: pipeline().CreatePipeline,
				}},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src"}, {Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "grants already in lifecycle.depends_on",
			resources: config.Resources{
				Schemas: map[string]*resources.Schema{"src": schema(grants("sp"))},
				Pipelines: map[string]*resources.Pipeline{"load": {
					BaseResource:   resources.BaseResource{Lifecycle: resources.Lifecycle{DependsOn: []resources.DependsOn{{Resource: "schemas.src.grants"}}}},
					CreatePipeline: pipeline().CreatePipeline,
				}},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "schemas.src.grants"}},
			},
		},
		{
			name: "pipeline depends on the permissions of a resource it refers to",
			resources: config.Resources{
				Jobs: map[string]*resources.Job{"job": job([]resources.JobPermission{{Level: "CAN_VIEW", GroupName: "users"}})},
				Pipelines: map[string]*resources.Pipeline{"load": {
					CreatePipeline: pipelines.CreatePipeline{Name: "${resources.jobs.job.name}"},
				}},
			},
			expected: map[string][]resources.DependsOn{
				"pipelines.load": {{Resource: "jobs.job.permissions"}},
			},
		},
		{
			name: "non-pipeline resource that refers to a resource with permissions gets nothing",
			resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job":    job([]resources.JobPermission{{Level: "CAN_VIEW", GroupName: "users"}}),
					"reader": {JobSettings: jobs.JobSettings{Name: "${resources.jobs.job.name}"}},
				},
			},
			expected: map[string][]resources.DependsOn{},
		},
		{
			name: "permissions that depend on the referring pipeline are skipped",
			resources: config.Resources{
				Jobs: map[string]*resources.Job{"job": job([]resources.JobPermission{{Level: "CAN_VIEW", UserName: "${resources.pipelines.load.name}"}})},
				Pipelines: map[string]*resources.Pipeline{"load": {
					CreatePipeline: pipelines.CreatePipeline{Name: "${resources.jobs.job.name}"},
				}},
			},
			expected: map[string][]resources.DependsOn{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{Config: config.Root{Resources: tc.resources}}

			// Normalization runs more than once, so applying the mutator again must not add anything.
			for range 2 {
				diags := bundle.Apply(t.Context(), b, CapturePipelineDependencies())
				require.NoError(t, diags.Error())

				actual := make(map[string][]resources.DependsOn)
				for _, group := range b.Config.Resources.AllResources() {
					for key, resource := range group.Resources {
						if reflect.ValueOf(resource).IsNil() {
							continue
						}
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
