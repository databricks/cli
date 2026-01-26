package tfdyn

import (
	"context"

	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

type postgresBranchConverter struct{}

func (c postgresBranchConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, diags := convert.Normalize(postgres.Branch{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "postgres branch normalization diagnostic: %s", diag.Summary)
	}

	vout, err := convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	out.PostgresBranch[key] = vout.AsAny()

	// TODO: Enable when PostgresBranchPermission is defined in Task 6
	// if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
	// 	permissions.PostgresBranchName = fmt.Sprintf("${databricks_postgres_branch.%s.name}", key)
	// 	out.Permissions["postgres_branch_"+key] = permissions
	// }

	return nil
}

func init() {
	registerConverter("postgres_branches", postgresBranchConverter{})
}
