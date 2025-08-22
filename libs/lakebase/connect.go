package lakebase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

func Connect(ctx context.Context, databaseInstanceName string, extraArgs ...string) error {
	cmdio.LogString(ctx, fmt.Sprintf("Connecting to Databricks Database Instance %s ...", databaseInstanceName))

	w := cmdctx.WorkspaceClient(ctx)

	// get user:
	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("error getting current user: %w", err)
	}

	// get database:
	db, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{
		Name: databaseInstanceName,
	})
	if err != nil {
		return fmt.Errorf("error getting Database Instance. Please confirm that database instance %s exists: %w", databaseInstanceName, err)
	}

	cmdio.LogString(ctx, "Postgres version: "+db.PgVersion)
	cmdio.LogString(ctx, fmt.Sprintf("Database instance status: %s", db.State))

	if db.State != database.DatabaseInstanceStateAvailable {
		if db.State == database.DatabaseInstanceStateStarting || db.State == database.DatabaseInstanceStateUpdating || db.State == database.DatabaseInstanceStateFailingOver {
			cmdio.LogString(ctx, "Please retry when the instance becomes available")
		}
		return errors.New("database instance is not ready for accepting connections")
	}

	// get credentials:
	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{databaseInstanceName},
		RequestId:     uuid.NewString(),
	})
	if err != nil {
		return fmt.Errorf("error getting database credentials: %w", err)
	}
	cmdio.LogString(ctx, "Successfully fetched database credentials")

	// Check if database name and port are already specified in extra arguments
	hasDbName := false
	hasPort := false
	for _, arg := range extraArgs {
		if arg == "-d" || strings.HasPrefix(arg, "--dbname=") {
			hasDbName = true
		}
		if arg == "-p" || strings.HasPrefix(arg, "--port=") {
			hasPort = true
		}
	}

	// Prepare command arguments
	args := []string{
		"psql",
		"--host=" + db.ReadWriteDns,
		"--username=" + user.UserName,
	}

	// Add default port only if not specified in extra arguments
	if !hasPort {
		args = append(args, "--port=5432")
	}

	// Add default database name only if not specified in extra arguments
	if !hasDbName {
		args = append(args, "--dbname=databricks_postgres")
	}

	// Append any extra arguments passed through
	args = append(args, extraArgs...)

	// Set environment variables for psql
	cmdEnv := append(os.Environ(),
		"PGPASSWORD="+cred.Token,
		"PGSSLMODE=require",
	)

	cmdio.LogString(ctx, fmt.Sprintf("Launching psql with connection to %s...", db.ReadWriteDns))

	// Execute psql command inline
	return exec.Execv(exec.ExecvOptions{
		Args: args,
		Env:  cmdEnv,
	})
}
