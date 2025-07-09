package lakebase

import (
	"context"
	"fmt"
	"os"

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
		fmt.Println("Error getting current user: ", err)
		return err
	}

	// get database:
	db, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{
		Name: databaseInstanceName,
	})
	if err != nil {
		fmt.Println("Error getting Database Instance: ", err)
		fmt.Println("Does the database instance exist?")
		return err
	}

	fmt.Println("Database status: ", db.State)
	fmt.Println("Database postgres version: ", db.PgVersion)

	// get credentials:
	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{databaseInstanceName},
		RequestId:     uuid.NewString(),
	})
	if err != nil {
		fmt.Println("Error getting database credentials: ", err)
		return err
	}
	fmt.Println("Successfully fetched database credentials")

	// Get current working directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		return err
	}

	// Prepare command arguments
	args := []string{
		"psql",
		"--host=" + db.ReadWriteDns,
		"--username=" + user.UserName,
		"--dbname=databricks_postgres",
		"--port=5432",
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
