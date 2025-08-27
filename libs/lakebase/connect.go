package lakebase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	execlib "github.com/databricks/cli/libs/exec"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

const (
	// Default retry configuration
	defaultMaxRetries    = 3
	defaultInitialDelay  = 1 * time.Second
	defaultMaxDelay      = 10 * time.Second
	defaultBackoffFactor = 2.0
)

// RetryConfig holds configuration for connection retry behavior
type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// tryPsqlInteractive launches psql interactively and returns an error if connection fails
func tryPsqlInteractive(ctx context.Context, args, env []string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		// Check if the error might be due to connection issues
		// Since we can't capture stderr when running interactively, we check the exit code
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode := exitError.ExitCode()

			// We do not use the Databricks SDK for checking whether the error is retryable because the call in question is not to the API
			// psql returns exit code 2 for connection failures
			if exitCode == 2 {
				return fmt.Errorf("connection failed (retryable): psql exited with code %d", exitCode)
			}
		}
		return err
	}

	return nil
}

func Connect(ctx context.Context, databaseInstanceName string, extraArgs ...string) error {
	return ConnectWithRetryConfig(ctx, databaseInstanceName, nil, extraArgs...)
}

func ConnectWithRetryConfig(ctx context.Context, databaseInstanceName string, retryConfig *RetryConfig, extraArgs ...string) error {
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

	// Use provided retry configuration or defaults
	if retryConfig == nil {
		retryConfig = &RetryConfig{
			MaxRetries:    defaultMaxRetries,
			InitialDelay:  defaultInitialDelay,
			MaxDelay:      defaultMaxDelay,
			BackoffFactor: defaultBackoffFactor,
		}
	}

	// If retries are disabled, go directly to interactive session
	if retryConfig.MaxRetries <= 0 {
		cmdio.LogString(ctx, fmt.Sprintf("Launching psql with connection to %s...", db.ReadWriteDns))
		return execlib.Execv(execlib.ExecvOptions{
			Args: args,
			Env:  cmdEnv,
		})
	}

	// Try launching psql with retry logic
	maxRetries := retryConfig.MaxRetries
	delay := retryConfig.InitialDelay

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			cmdio.LogString(ctx, fmt.Sprintf("Connection attempt %d/%d failed, retrying in %v...", attempt, maxRetries+1, delay))

			// Wait with context cancellation support
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * retryConfig.BackoffFactor)
			if delay > retryConfig.MaxDelay {
				delay = retryConfig.MaxDelay
			}
		}

		cmdio.LogString(ctx, fmt.Sprintf("Launching psql session to %s (attempt %d/%d)...", db.ReadWriteDns, attempt+1, maxRetries+1))

		// Try to launch psql and capture the exit status
		err := tryPsqlInteractive(ctx, args, cmdEnv)
		if err == nil {
			// psql exited normally (user quit)
			return nil
		}

		lastErr = err

		// Check if this is a retryable error
		// We do not use the Databricks SDK for checking whether the error is retryable because the call in question is not to the API
		if !strings.Contains(err.Error(), "connection failed (retryable)") {
			// Non-retryable error, fail immediately
			return err
		}

		if attempt < maxRetries {
			cmdio.LogString(ctx, fmt.Sprintf("Connection failed with retryable error: %v", err))
		}
	}

	// All retries exhausted
	return fmt.Errorf("failed to connect after %d attempts, last error: %w", maxRetries+1, lastErr)
}
