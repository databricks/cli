package psql

import (
	"context"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	lakebasepsql "github.com/databricks/cli/libs/lakebase/psql"
	lakebasev1 "github.com/databricks/cli/libs/lakebase/v1"
)

// connectProvisioned connects to a Lakebase Provisioned database instance.
func connectProvisioned(ctx context.Context, instanceName string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	db, err := lakebasev1.GetDatabaseInstance(ctx, w, instanceName)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Instance: "+db.Name+" (provisioned)")
	return lakebasev1.Connect(ctx, w, db, retryConfig, extraArgs...)
}
