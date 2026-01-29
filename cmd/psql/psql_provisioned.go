package psql

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	lakebasepsql "github.com/databricks/cli/libs/lakebase/psql"
	lakebasev1 "github.com/databricks/cli/libs/lakebase/v1"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

// connectProvisioned connects to a Lakebase Provisioned database instance.
// If instanceName is empty, prompts the user to select one.
func connectProvisioned(ctx context.Context, instanceName string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	instance, err := resolveInstance(ctx, w, instanceName)
	if err != nil {
		return err
	}

	return lakebasev1.Connect(ctx, w, instance, retryConfig, extraArgs...)
}

// resolveInstance resolves an instance name to a full instance object.
// If instanceName is empty, prompts the user to select one.
func resolveInstance(ctx context.Context, w *databricks.WorkspaceClient, instanceName string) (*database.DatabaseInstance, error) {
	// If instance not specified, select one
	if instanceName == "" {
		var err error
		instanceName, err = selectInstanceID(ctx, w)
		if err != nil {
			return nil, err
		}
	}

	instance, err := lakebasev1.GetDatabaseInstance(ctx, w, instanceName)
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "Instance: "+instance.Name)
	return instance, nil
}

// selectInstanceID auto-selects if there's only one instance, otherwise prompts user to select.
// Returns the instance name.
func selectInstanceID(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading instances...")
	instances, err := w.Database.ListDatabaseInstancesAll(ctx, database.ListDatabaseInstancesRequest{})
	sp.Close()
	if err != nil {
		return "", err
	}

	if len(instances) == 0 {
		return "", errors.New("no database instances found in workspace")
	}

	// Auto-select if there's only one instance
	if len(instances) == 1 {
		return instances[0].Name, nil
	}

	// Multiple instances, prompt user to select
	var items []cmdio.Tuple
	for _, inst := range instances {
		items = append(items, cmdio.Tuple{Name: inst.Name, Id: inst.Name})
	}

	return cmdio.SelectOrdered(ctx, items, "Select instance")
}
