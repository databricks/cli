package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/listing"
	libpsql "github.com/databricks/cli/libs/psql"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	database "github.com/databricks/sdk-go/database/v1"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
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

	user, err := w.Scim.Me(ctx, sdkscim.MeRequest{})
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	if instance.State != database.DatabaseInstance_State_Available {
		cmdio.LogString(ctx, fmt.Sprintf("Instance status: %s", instance.State))
		if instance.State == database.DatabaseInstance_State_Starting || instance.State == database.DatabaseInstance_State_Updating || instance.State == database.DatabaseInstance_State_FailingOver {
			cmdio.LogString(ctx, "Please retry when the instance becomes available")
		}
		return errors.New("database instance is not ready for accepting connections")
	}

	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{utils.Deref(instance.Name)},
		RequestId:     new(uuid.NewString()),
	})
	if err != nil {
		return fmt.Errorf("failed to get database credentials: %w", err)
	}

	cmdio.LogString(ctx, "Connecting to database instance...")

	return libpsql.Connect(ctx, libpsql.ConnectOptions{
		Host:            utils.Deref(instance.ReadWriteDns),
		Username:        utils.Deref(user.UserName),
		Password:        utils.Deref(cred.Token),
		DefaultDatabase: provisionedDefaultDatabase,
		ExtraArgs:       extraArgs,
	}, retryConfig)
}

// resolveInstance resolves an instance name to a full instance object.
// If instanceName is empty, prompts the user to select one.
func resolveInstance(ctx context.Context, w *sdkclients.Clients, instanceName string) (*database.DatabaseInstance, error) {
	// If instance not specified, select one
	if instanceName == "" {
		var err error
		instanceName, err = selectInstanceID(ctx, w)
		if err != nil {
			return nil, err
		}
	}

	instance, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{
		Name: new(instanceName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	// Ensure Name is set (API response may not include it)
	if utils.Deref(instance.Name) == "" {
		instance.Name = new(instanceName)
	}

	cmdio.LogString(ctx, "Instance: "+utils.Deref(instance.Name))
	return instance, nil
}

// selectInstanceID auto-selects if there's only one instance, otherwise prompts user to select.
// Returns the instance name.
func selectInstanceID(ctx context.Context, w *sdkclients.Clients) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading instances...")
	instances, err := listing.Collect(w.Database.ListDatabaseInstancesIter(ctx, database.ListDatabaseInstancesRequest{}))
	sp.Close()
	if err != nil {
		return "", err
	}

	if len(instances) == 0 {
		return "", errors.New("no Lakebase Provisioned instances found in workspace")
	}

	// Auto-select if there's only one instance
	if len(instances) == 1 {
		return utils.Deref(instances[0].Name), nil
	}

	// Multiple instances, prompt user to select
	var items []cmdio.Tuple
	for _, inst := range instances {
		items = append(items, cmdio.Tuple{Name: utils.Deref(inst.Name), Id: utils.Deref(inst.Name)})
	}

	return cmdio.SelectOrdered(ctx, items, "Select instance")
}
