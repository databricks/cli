package lakebase

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/exec"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

func Connect(ctx context.Context, databaseInstanceName string, extraArgs ...string) error {
	fmt.Printf("Connecting to Databricks Database Instance %s ...\n", databaseInstanceName)

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

	fmt.Println("Database status: ", db.State)
	fmt.Println("Database postgres version: ", db.PgVersion)

	// get credentials:
	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{databaseInstanceName},
		RequestId:     uuid.NewString(),
	})
	if err != nil {
		return fmt.Errorf("error getting database credentials: %w", err)
	}
	fmt.Println("Successfully fetched database credentials")

	// Get current working directory
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}

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

	fmt.Printf("Launching psql with connection to %s...\n", db.ReadWriteDns)

	// Execute psql command inline
	return exec.Execv(exec.ExecvOptions{
		Args: args,
		Env:  cmdEnv,
		Dir:  dir,
	})
}
