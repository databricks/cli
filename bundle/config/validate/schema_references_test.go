package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestValidateSchemaReferencesForPipelines(t *testing.T) {
	pipelineTargetL := dyn.Location{File: "file1", Line: 1, Column: 1}
	pipelineSchemaL := dyn.Location{File: "file2", Line: 2, Column: 2}
	pipelineL := dyn.Location{File: "file3", Line: 3, Column: 3}
	schemaL := dyn.Location{File: "file4", Line: 4, Column: 4}

	for _, tc := range []struct {
		schemaV  string
		targetV  string
		catalogV string
		want     diag.Diagnostics
	}{
		{
			schemaV:  "",
			targetV:  "",
			catalogV: "",
			want:     diag.Diagnostics{},
		},
		{
			schemaV:  "",
			targetV:  "",
			catalogV: "main",
			want: diag.Diagnostics{{
				Summary:  "Unity Catalog pipeline should have a schema or target defined",
				Severity: diag.Error,
				Detail: `The target or schema field is required for UC pipelines. Reason: DLT
requires specifying a target schema for UC pipelines. Please use the
TEMPORARY keyword in the CREATE MATERIALIZED VIEW or CREATE STREAMING
TABLE statement if you do not wish to publish your dataset.`,
				Locations: []dyn.Location{pipelineL},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.pipelines.p1.schema"),
					dyn.MustPathFromString("resources.pipelines.p1.target"),
				},
			}},
		},
		{
			schemaV:  "both",
			targetV:  "both",
			catalogV: "main",
			want: diag.Diagnostics{{
				Severity:  diag.Error,
				Summary:   "Both schema and target are defined in a Unity Catalog pipeline. Only one of them should be defined.",
				Locations: []dyn.Location{pipelineSchemaL, pipelineTargetL},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.pipelines.p1.schema"),
					dyn.MustPathFromString("resources.pipelines.p1.target"),
				},
			}},
		},
		{
			schemaV:  "schema1",
			targetV:  "",
			catalogV: "other",
			want:     diag.Diagnostics{},
		},
		{
			schemaV:  "schema1",
			targetV:  "",
			catalogV: "main",
			want: diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  `Use ${resources.schemas.s1.name} syntax to refer to the UC schema instead of directly using its name "schema1"`,
				Detail: `Using ${resources.schemas.s1.name} will allow DABs to capture the deploy time dependency this DLT pipeline
has on the schema "schema1" and deploy changes to the schema before deploying the pipeline.`,
				Locations: []dyn.Location{pipelineSchemaL, schemaL},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.pipelines.p1.schema"),
					dyn.MustPathFromString("resources.schemas.s1"),
				},
			}},
		},
		{
			schemaV:  "",
			targetV:  "schema1",
			catalogV: "main",
			want: diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  `Use ${resources.schemas.s1.name} syntax to refer to the UC schema instead of directly using its name "schema1"`,
				Detail: `Using ${resources.schemas.s1.name} will allow DABs to capture the deploy time dependency this DLT pipeline
has on the schema "schema1" and deploy changes to the schema before deploying the pipeline.`,
				Locations: []dyn.Location{pipelineTargetL, schemaL},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.pipelines.p1.target"),
					dyn.MustPathFromString("resources.schemas.s1"),
				},
			}},
		},
		{
			schemaV:  "${resources.schemas.s1.name}",
			targetV:  "",
			catalogV: "main",
			want:     diag.Diagnostics{},
		},
		{
			schemaV:  "",
			targetV:  "${resources.schemas.s1.name}",
			catalogV: "main",
			want:     diag.Diagnostics{},
		},
	} {

		b := &bundle.Bundle{
			Config: config.Root{
				Resources: config.Resources{
					Schemas: map[string]*resources.Schema{
						"s1": {
							CreateSchema: &catalog.CreateSchema{
								CatalogName: "main",
								Name:        "schema1",
							},
						},
					},
					Pipelines: map[string]*resources.Pipeline{
						"p1": {
							PipelineSpec: &pipelines.PipelineSpec{
								Name:    "abc",
								Schema:  tc.schemaV,
								Target:  tc.targetV,
								Catalog: tc.catalogV,
							},
						},
					},
				},
			},
		}

		bundletest.SetLocation(b, "resources.schemas.s1", []dyn.Location{schemaL})
		bundletest.SetLocation(b, "resources.pipelines.p1", []dyn.Location{pipelineL})
		if tc.schemaV != "" {
			bundletest.SetLocation(b, "resources.pipelines.p1.schema", []dyn.Location{pipelineSchemaL})
		}
		if tc.targetV != "" {
			bundletest.SetLocation(b, "resources.pipelines.p1.target", []dyn.Location{pipelineTargetL})
		}

		diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), SchemaReferences())
		assert.Equal(t, tc.want, diags)
	}
}

func TestValidateSchemaReferencesForVolumes(t *testing.T) {
	schemaL := dyn.Location{File: "file1", Line: 1, Column: 1}
	volumeSchemaL := dyn.Location{File: "file2", Line: 2, Column: 2}
	for _, tc := range []struct {
		catalogV string
		schemaV  string
		want     diag.Diagnostics
	}{
		{
			catalogV: "main",
			schemaV:  "schema1",
			want: diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  `Use ${resources.schemas.s1.name} syntax to refer to the UC schema instead of directly using its name "schema1"`,
				Detail: `Using ${resources.schemas.s1.name} will allow DABs to capture the deploy time dependency this Volume
has on the schema "schema1" and deploy changes to the schema before deploying the Volume.`,
				Locations: []dyn.Location{schemaL, volumeSchemaL},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.volumes.v1.schema"),
					dyn.MustPathFromString("resources.schemas.s1"),
				},
			}},
		},
		{
			catalogV: "main",
			schemaV:  "${resources.schemas.s1.name}",
			want:     diag.Diagnostics{},
		},
		{
			catalogV: "main",
			schemaV:  "other",
			want:     diag.Diagnostics{},
		},
		{
			catalogV: "other",
			schemaV:  "schema1",
			want:     diag.Diagnostics{},
		},
	} {
		b := bundle.Bundle{
			Config: config.Root{
				Resources: config.Resources{
					Schemas: map[string]*resources.Schema{
						"s1": {
							CreateSchema: &catalog.CreateSchema{
								CatalogName: "main",
								Name:        "schema1",
							},
						},
					},
					Volumes: map[string]*resources.Volume{
						"v1": {
							CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
								SchemaName:  tc.schemaV,
								CatalogName: tc.catalogV,
								Name:        "my_volume",
							},
						},
					},
				},
			},
		}

		bundletest.SetLocation(&b, "resources.schemas.s1", []dyn.Location{schemaL})
		bundletest.SetLocation(&b, "resources.volumes.v1.schema_name", []dyn.Location{volumeSchemaL})

		diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(&b), SchemaReferences())
		assert.Equal(t, tc.want, diags)
	}
}
