package lakebasev1

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/lakebase/psql"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

// GetDatabaseInstance retrieves a database instance by name.
func GetDatabaseInstance(ctx context.Context, w *databricks.WorkspaceClient, name string) (*database.DatabaseInstance, error) {
	db, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting Database Instance. Please confirm that database instance %s exists: %w", name, err)
	}
	// Ensure Name is set (API response may not include it)
	if db.Name == "" {
		db.Name = name
	}
	return db, nil
}

// Connect connects to a database instance with retry logic.
func Connect(ctx context.Context, w *databricks.WorkspaceClient, db *database.DatabaseInstance, retryConfig psql.RetryConfig, extraArgs ...string) error {
	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("error getting current user: %w", err)
	}

	if db.State != database.DatabaseInstanceStateAvailable {
		cmdio.LogString(ctx, fmt.Sprintf("Instance status: %s", db.State))
		if db.State == database.DatabaseInstanceStateStarting || db.State == database.DatabaseInstanceStateUpdating || db.State == database.DatabaseInstanceStateFailingOver {
			cmdio.LogString(ctx, "Please retry when the instance becomes available")
		}
		return errors.New("database instance is not ready for accepting connections")
	}

	cmdio.LogString(ctx, "Connecting...")

	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{db.Name},
		RequestId:     uuid.NewString(),
	})
	if err != nil {
		return fmt.Errorf("error getting database credentials: %w", err)
	}

	return psql.Connect(ctx, psql.ConnectOptions{
		Host:            db.ReadWriteDns,
		Username:        user.UserName,
		Password:        cred.Token,
		DefaultDatabase: "databricks_postgres",
		ExtraArgs:       extraArgs,
	}, retryConfig)
}
