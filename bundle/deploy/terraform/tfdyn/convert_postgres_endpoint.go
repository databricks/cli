package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type postgresEndpointConverter struct{}

func (c postgresEndpointConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, diags := convert.Normalize(postgres.Endpoint{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "postgres endpoint normalization diagnostic: %s", diag.Summary)
	}

	vout, err := convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	out.PostgresEndpoint[key] = vout.AsAny()

	// TODO: Enable when PostgresEndpointPermission is defined in Task 6
	// if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
	// 	permissions.PostgresEndpointName = fmt.Sprintf("${databricks_postgres_endpoint.%s.name}", key)
	// 	out.Permissions["postgres_endpoint_"+key] = permissions
	// }

	return nil
}

func init() {
	registerConverter("postgres_endpoints", postgresEndpointConverter{})
}
