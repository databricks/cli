package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

type postgresProjectConverter struct{}

func (c postgresProjectConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// The bundle config has flattened ProjectSpec fields at the top level.
	// Terraform expects them nested in a "spec" block. We need to restructure:
	// - Keep project_id at top level
	// - Move display_name, pg_version, history_retention_duration, default_endpoint_settings into spec

	specFields := []string{"display_name", "pg_version", "history_retention_duration", "default_endpoint_settings"}

	// Build the spec block from the flattened fields
	specMap := make(map[string]dyn.Value)
	for _, field := range specFields {
		if v := vin.Get(field); v.Kind() != dyn.KindInvalid {
			specMap[field] = v
		}
	}

	// Build the output with project_id and spec
	outMap := make(map[string]dyn.Value)

	// Keep project_id at top level
	if v := vin.Get("project_id"); v.Kind() != dyn.KindInvalid {
		outMap["project_id"] = v
	}

	// Add spec block if we have any spec fields
	if len(specMap) > 0 {
		outMap["spec"] = dyn.V(specMap)
	}

	vout := dyn.V(outMap)

	// Normalize the output value to the Terraform schema.
	vout, diags := convert.Normalize(schema.ResourcePostgresProject{}, vout)
	for _, diag := range diags {
		log.Debugf(ctx, "postgres project normalization diagnostic: %s", diag.Summary)
	}

	vout, err := convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	out.PostgresProject[key] = vout.AsAny()

	// TODO: Enable permissions in Task 6
	// Configure permissions for this resource.
	// if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
	// 	permissions.PostgresProjectName = fmt.Sprintf("${databricks_postgres_project.%s.name}", key)
	// 	out.Permissions["postgres_project_"+key] = permissions
	// }

	return nil
}

func init() {
	registerConverter("postgres_projects", postgresProjectConverter{})
}
