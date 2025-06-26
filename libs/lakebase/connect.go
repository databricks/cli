package lakebase

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

func Connect(ctx context.Context, databaseInstanceName string) error {
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

	// Prepare psql command with connection parameters
	cmd := exec.CommandContext(ctx, "psql",
		"--host="+db.ReadWriteDns,
		"--username="+user.UserName,
		"--dbname=databricks_postgres",
		"--port=5432",
	)

	// Set environment variables for psql
	cmd.Env = append(os.Environ(),
		"PGPASSWORD="+cred.Token,
		"PGSSLMODE=require",
	)

	// Connect stdin, stdout, and stderr to enable interactive session
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Launching psql with connection to %s...\n", db.ReadWriteDns)

	// Execute psql command
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error running psql: %v\n", err)
		fmt.Println("Do you have `psql` installed in your path?")
		return err
	}

	return nil
}
