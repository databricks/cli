package lakebasev2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	lakebasev1 "github.com/databricks/cli/libs/lakebase/v1"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/execv"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// ExtractIDFromName extracts the ID component from a resource name.
// For example, ExtractIDFromName("projects/foo/branches/bar", "branches") returns "bar".
func ExtractIDFromName(name, component string) string {
	parts := strings.Split(name, "/")
	for i := range len(parts) - 1 {
		if parts[i] == component {
			return parts[i+1]
		}
	}
	return name
}

// GetEndpoint retrieves an endpoint by its full resource name.
func GetEndpoint(ctx context.Context, w *databricks.WorkspaceClient, projectID, branchID, endpointID string) (*postgres.Endpoint, error) {
	endpointName := fmt.Sprintf("projects/%s/branches/%s/endpoints/%s", projectID, branchID, endpointID)
	endpoint, err := w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{
		Name: endpointName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint %s: %w", endpointName, err)
	}
	return endpoint, nil
}

// ConnectWithRetryConfig connects to a Postgres endpoint with retry logic.
func ConnectWithRetryConfig(ctx context.Context, endpoint *postgres.Endpoint, retryConfig lakebasev1.RetryConfig, extraArgs ...string) error {
	endpointID := ExtractIDFromName(endpoint.Name, "endpoints")
	cmdio.LogString(ctx, fmt.Sprintf("Connecting to Postgres endpoint %s ...", endpointID))

	w := cmdctx.WorkspaceClient(ctx)

	// Get current user
	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("error getting current user: %w", err)
	}

	// Check endpoint state
	if endpoint.Status == nil {
		return errors.New("endpoint status is not available")
	}

	state := endpoint.Status.CurrentState
	cmdio.LogString(ctx, fmt.Sprintf("Endpoint state: %s", state))

	if state != postgres.EndpointStatusStateActive && state != postgres.EndpointStatusStateIdle {
		if state == postgres.EndpointStatusStateInit {
			cmdio.LogString(ctx, "Please retry when the endpoint becomes active")
		}
		return errors.New("endpoint is not ready for accepting connections")
	}

	if endpoint.Status.Hosts == nil || endpoint.Status.Hosts.Host == "" {
		return errors.New("endpoint host information is not available")
	}
	host := endpoint.Status.Hosts.Host

	// Generate credentials
	cred, err := w.Postgres.GenerateDatabaseCredential(ctx, postgres.GenerateDatabaseCredentialRequest{
		Endpoint: endpoint.Name,
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
		"--host=" + host,
		"--username=" + user.UserName,
	}

	// Add default port only if not specified in extra arguments
	if !hasPort {
		args = append(args, "--port=5432")
	}

	// Add default database name only if not specified in extra arguments
	if !hasDbName {
		args = append(args, "--dbname=postgres")
	}

	// Append any extra arguments passed through
	args = append(args, extraArgs...)

	// Set environment variables for psql
	cmdEnv := append(os.Environ(),
		"PGPASSWORD="+cred.Token,
		"PGSSLMODE=require",
	)

	// If retries are disabled, go directly to interactive session
	if retryConfig.MaxRetries <= 0 {
		cmdio.LogString(ctx, fmt.Sprintf("Launching psql with connection to %s...", host))
		return execv.Execv(execv.Options{
			Args: args,
			Env:  cmdEnv,
		})
	}

	// Try launching psql with retry logic
	maxRetries := retryConfig.MaxRetries
	delay := retryConfig.InitialDelay

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			cmdio.LogString(ctx, fmt.Sprintf("Connection attempt %d/%d failed, retrying in %v...", attempt, maxRetries, delay))

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

		cmdio.LogString(ctx, fmt.Sprintf("Launching psql session to %s (attempt %d/%d)...", host, attempt+1, maxRetries))

		// Try to launch psql and capture the exit status
		err := attemptConnection(ctx, args, cmdEnv)
		if err == nil {
			// psql exited normally (user quit)
			return nil
		}

		lastErr = err

		// Check if this is a retryable error
		if !strings.Contains(err.Error(), "connection failed (retryable)") {
			// Non-retryable error, fail immediately
			return err
		}

		if attempt < maxRetries {
			cmdio.LogString(ctx, fmt.Sprintf("Connection failed with retryable error: %v", err))
		}
	}

	// All retries exhausted
	return fmt.Errorf("failed to connect after %d attempts, last error: %w", maxRetries, lastErr)
}

// attemptConnection launches psql interactively and returns an error if connection fails.
func attemptConnection(ctx context.Context, args, env []string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode := exitError.ExitCode()
			// psql returns exit code 2 for connection failures
			if exitCode == 2 {
				return fmt.Errorf("connection failed (retryable): psql exited with code %d", exitCode)
			}
		}
	}

	return err
}
