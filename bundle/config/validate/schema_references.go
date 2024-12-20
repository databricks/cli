package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// Validate that any references to UC schemas defined in the DAB use the ${resources.schemas...}
// syntax to capture the deploy time dependency.
func SchemaReferences() bundle.ReadOnlyMutator {
	return &schemaReferences{}
}

type schemaReferences struct{}

func (v *schemaReferences) Name() string {
	return "validate:schema_dependency"
}

func findSchemaInBundle(rb bundle.ReadOnlyBundle, catalogName, schemaName string) ([]dyn.Location, dyn.Path, bool) {
	for k, s := range rb.Config().Resources.Schemas {
		if s.CatalogName != catalogName || s.Name != schemaName {
			continue
		}
		return rb.Config().GetLocations("resources.schemas." + k), dyn.NewPath(dyn.Key("resources"), dyn.Key("schemas"), dyn.Key(k)), true
	}
	return nil, nil, false
}

func (v *schemaReferences) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}
	for k, p := range rb.Config().Resources.Pipelines {
		// Skip if the pipeline uses hive metastore. The DLT API allows creating
		// a pipeline without a schema or target when using hive metastore.
		if p.Catalog == "" {
			continue
		}

		schemaName := ""
		fieldPath := dyn.Path{}
		schemaLocation := []dyn.Location{}
		switch {
		case p.Schema == "" && p.Target == "":
			// The error message is identical to the one DLT backend returns when
			// a schema is not defined for a UC DLT pipeline (date: 20 Dec 2024).
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unity Catalog pipeline should have a schema or target defined",
				Detail: `The target or schema field is required for UC pipelines. Reason: DLT
requires specifying a target schema for UC pipelines. Please use the
TEMPORARY keyword in the CREATE MATERIALIZED VIEW or CREATE STREAMING
TABLE statement if you do not wish to publish your dataset.`,
				Locations: rb.Config().GetLocations("resources.pipelines." + k),
				Paths: []dyn.Path{
					dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(k), dyn.Key("schema")),
					dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(k), dyn.Key("target")),
				},
			})
			continue
		case p.Schema != "" && p.Target != "":
			locations := rb.Config().GetLocations("resources.pipelines." + k + ".schema")
			locations = append(locations, rb.Config().GetLocations("resources.pipelines."+k+".target")...)

			// The Databricks Terraform provider already has client side validation
			// that does not allow this today. Having this here allows us to float
			// this validation on `bundle validate` and provide location information.
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Both schema and target are defined in a Unity Catalog pipeline. Only one of them should be defined.",
				Locations: locations,
				Paths: []dyn.Path{
					dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(k), dyn.Key("schema")),
					dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(k), dyn.Key("target")),
				},
			})
			continue
		case p.Schema != "":
			schemaName = p.Schema
			fieldPath = dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(k), dyn.Key("schema"))
			schemaLocation = rb.Config().GetLocations("resources.pipelines." + k + ".schema")
		case p.Target != "":
			schemaName = p.Target
			fieldPath = dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(k), dyn.Key("target"))
			schemaLocation = rb.Config().GetLocations("resources.pipelines." + k + ".target")
		}

		// Check if the schema is defined in the bundle.
		matchLocations, matchPath, found := findSchemaInBundle(rb, p.Catalog, schemaName)
		if !found {
			continue
		}

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Use ${%s.name} syntax to refer to the UC schema instead of directly using its name %q", matchPath, schemaName),
			Detail: fmt.Sprintf(`Using ${%s.name} will allow DABs to capture the deploy time dependency this DLT pipeline
has on the schema %q and deploy changes to the schema before deploying the pipeline.`, matchPath, schemaName),
			Locations: append(schemaLocation, matchLocations...),
			Paths: []dyn.Path{
				fieldPath,
				matchPath,
			},
		})
	}

	for k, v := range rb.Config().Resources.Volumes {
		if v.CatalogName == "" || v.SchemaName == "" {
			continue
		}

		matchLocations, matchPath, found := findSchemaInBundle(rb, v.CatalogName, v.SchemaName)
		if !found {
			continue
		}

		fieldLocations := rb.Config().GetLocations("resources.volumes." + k + ".schema_name")

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Use ${%s.name} syntax to refer to the UC schema instead of directly using its name %q", matchPath, v.SchemaName),
			Detail: fmt.Sprintf(`Using ${%s.name} will allow DABs to capture the deploy time dependency this Volume
has on the schema %q and deploy changes to the schema before deploying the Volume.`, matchPath, v.SchemaName),
			Locations: append(matchLocations, fieldLocations...),
			Paths: []dyn.Path{
				dyn.NewPath(dyn.Key("resources"), dyn.Key("volumes"), dyn.Key(k), dyn.Key("schema")),
				matchPath,
			},
		})
	}

	return diags
}
