package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	libpsql "github.com/databricks/cli/libs/psql"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

// provisionedDefaultDatabase is the default database for Lakebase Provisioned instances.
const provisionedDefaultDatabase = "databricks_postgres"

// connectProvisioned connects to a Lakebase Provisioned database instance.
// If instanceName is empty, prompts the user to select one.
func connectProvisioned(ctx context.Context, instanceName string, retryConfig libpsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	instance, err := resolveInstance(ctx, w, instanceName)
	if err != nil {
		return err
	}

	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	if instance.State != database.DatabaseInstanceStateAvailable {
		cmdio.LogString(ctx, fmt.Sprintf("Instance status: %s", instance.State))
		if instance.State == database.DatabaseInstanceStateStarting || instance.State == database.DatabaseInstanceStateUpdating || instance.State == database.DatabaseInstanceStateFailingOver {
			cmdio.LogString(ctx, "Please retry when the instance becomes available")
		}
		return errors.New("database instance is not ready for accepting connections")
	}

	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{instance.Name},
		RequestId:     uuid.NewString(),
	})
	if err != nil {
		return fmt.Errorf("failed to get database credentials: %w", err)
	}

	cmdio.LogString(ctx, "Connecting to database instance...")

	return libpsql.Connect(ctx, libpsql.ConnectOptions{
		Host:            instance.ReadWriteDns,
		Username:        user.UserName,
		Password:        cred.Token,
		DefaultDatabase: provisionedDefaultDatabase,
		ExtraArgs:       extraArgs,
	}, retryConfig)
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

	instance, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{
		Name: instanceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	// Ensure Name is set (API response may not include it)
	if instance.Name == "" {
		instance.Name = instanceName
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
		return "", errors.New("no Lakebase Provisioned instances found in workspace")
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
