package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/lakebase/target"
	libpsql "github.com/databricks/cli/libs/psql"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
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

	token, err := target.ProvisionedCredential(ctx, w, instance.Name)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Connecting to database instance...")

	return libpsql.Connect(ctx, libpsql.ConnectOptions{
		Host:            instance.ReadWriteDns,
		Username:        user.UserName,
		Password:        token,
		DefaultDatabase: provisionedDefaultDatabase,
		ExtraArgs:       extraArgs,
	}, retryConfig)
}

// resolveInstance resolves an instance name to a full instance object.
// If instanceName is empty, prompts the user to select one.
func resolveInstance(ctx context.Context, w *databricks.WorkspaceClient, instanceName string) (*database.DatabaseInstance, error) {
	if instanceName == "" {
		var err error
		instanceName, err = selectInstanceID(ctx, w)
		if err != nil {
			return nil, err
		}
	}

	instance, err := target.GetProvisioned(ctx, w, instanceName)
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
	id, err := target.AutoSelectProvisioned(ctx, w)
	sp.Close()

	var amb *target.AmbiguousError
	if !errors.As(err, &amb) {
		return id, err
	}
	return selectAmbiguous(ctx, amb, "Select instance")
}
