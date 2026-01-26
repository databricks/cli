package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type postgresProjectConverter struct{}

func (c postgresProjectConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(postgres.Project{}, vin)
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
