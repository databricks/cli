package lakebasev2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	lakebasev1 "github.com/databricks/cli/libs/lakebase/v1"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/execv"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// parseResourcePath extracts project, branch, and endpoint IDs from a resource path.
// Supported formats:
//   - "projects/{project_id}" -> project only
//   - "projects/{project_id}/branches/{branch_id}" -> project and branch
//   - "projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}" -> all three
func parseResourcePath(input string) (project, branch, endpoint string) {
	// Match full endpoint path
	endpointRe := regexp.MustCompile(`^projects/([^/]+)/branches/([^/]+)/endpoints/([^/]+)$`)
	if matches := endpointRe.FindStringSubmatch(input); len(matches) == 4 {
		return matches[1], matches[2], matches[3]
	}

	// Match branch path
	branchRe := regexp.MustCompile(`^projects/([^/]+)/branches/([^/]+)$`)
	if matches := branchRe.FindStringSubmatch(input); len(matches) == 3 {
		return matches[1], matches[2], ""
	}

	// Match project path
	projectRe := regexp.MustCompile(`^projects/([^/]+)$`)
	if matches := projectRe.FindStringSubmatch(input); len(matches) == 2 {
		return matches[1], "", ""
	}

	return "", "", ""
}

// ResolveEndpoint resolves a partial specification to a full endpoint.
// Uses interactive selection when components are missing.
func ResolveEndpoint(ctx context.Context, w *databricks.WorkspaceClient,
	projectID, branchID, endpointID string) (*postgres.Endpoint, error) {

	// Build the full resource name
	projectName := fmt.Sprintf("projects/%s", projectID)

	// If branch not specified, select one
	if branchID == "" {
		branch, err := selectBranch(ctx, w, projectName)
		if err != nil {
			return nil, fmt.Errorf("failed to select branch: %w", err)
		}
		// Extract branch ID from the branch name
		branchID = extractIDFromName(branch.Name, "branches")
	}

	branchName := fmt.Sprintf("%s/branches/%s", projectName, branchID)

	// If endpoint not specified, select one
	if endpointID == "" {
		endpoint, err := selectEndpoint(ctx, w, branchName)
		if err != nil {
			return nil, fmt.Errorf("failed to select endpoint: %w", err)
		}
		return endpoint, nil
	}

	endpointName := fmt.Sprintf("%s/endpoints/%s", branchName, endpointID)
	endpoint, err := w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{
		Name: endpointName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint %s: %w", endpointName, err)
	}
	return endpoint, nil
}

// selectBranch auto-selects if there's only one branch, otherwise prompts user to select.
func selectBranch(ctx context.Context, w *databricks.WorkspaceClient, projectName string) (*postgres.Branch, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading branches...")
	branches, err := w.Postgres.ListBranchesAll(ctx, postgres.ListBranchesRequest{
		Parent: projectName,
	})
	sp.Close()
	if err != nil {
		return nil, err
	}

	if len(branches) == 0 {
		return nil, errors.New("no branches found in project")
	}

	// Auto-select if there's only one branch
	if len(branches) == 1 {
		branchID := extractIDFromName(branches[0].Name, "branches")
		cmdio.LogString(ctx, "Selected branch: "+branchID)
		return &branches[0], nil
	}

	// Multiple branches, prompt user to select
	var items []cmdio.Tuple
	for _, branch := range branches {
		branchID := extractIDFromName(branch.Name, "branches")
		items = append(items, cmdio.Tuple{Name: branchID, Id: branchID})
	}

	selectedID, err := cmdio.SelectOrdered(ctx, items, "Select branch")
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "Selected branch: "+selectedID)

	// Find the selected branch by matching the ID
	for i := range branches {
		if extractIDFromName(branches[i].Name, "branches") == selectedID {
			return &branches[i], nil
		}
	}

	return nil, errors.New("selected branch not found")
}

// selectEndpoint auto-selects if there's only one endpoint, otherwise prompts user to select.
func selectEndpoint(ctx context.Context, w *databricks.WorkspaceClient, branchName string) (*postgres.Endpoint, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading endpoints...")
	endpoints, err := w.Postgres.ListEndpointsAll(ctx, postgres.ListEndpointsRequest{
		Parent: branchName,
	})
	sp.Close()
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, errors.New("no endpoints found in branch")
	}

	// Auto-select if there's only one endpoint
	if len(endpoints) == 1 {
		endpointID := extractIDFromName(endpoints[0].Name, "endpoints")
		cmdio.LogString(ctx, "Selected endpoint: "+endpointID)
		return &endpoints[0], nil
	}

	// Multiple endpoints, prompt user to select
	var items []cmdio.Tuple
	for _, endpoint := range endpoints {
		endpointID := extractIDFromName(endpoint.Name, "endpoints")
		items = append(items, cmdio.Tuple{Name: endpointID, Id: endpointID})
	}

	selectedID, err := cmdio.SelectOrdered(ctx, items, "Select endpoint")
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, fmt.Sprintf("Selected endpoint: %s", selectedID))

	// Find the selected endpoint by matching the ID
	for i := range endpoints {
		if extractIDFromName(endpoints[i].Name, "endpoints") == selectedID {
			return &endpoints[i], nil
		}
	}

	return nil, errors.New("selected endpoint not found")
}

// extractIDFromName extracts the ID component from a resource name.
// For example, extractIDFromName("projects/foo/branches/bar", "branches") returns "bar".
func extractIDFromName(name, component string) string {
	parts := strings.Split(name, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == component {
			return parts[i+1]
		}
	}
	return name
}

// ConnectWithRetryConfig connects to a Postgres endpoint with retry logic.
func ConnectWithRetryConfig(ctx context.Context, endpoint *postgres.Endpoint, retryConfig lakebasev1.RetryConfig, extraArgs ...string) error {
	endpointID := extractIDFromName(endpoint.Name, "endpoints")
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
