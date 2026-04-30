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
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// autoscalingDefaultDatabase is the default database for Lakebase Autoscaling projects.
const autoscalingDefaultDatabase = "databricks_postgres"

// connectAutoscaling connects to a Lakebase Autoscaling endpoint.
func connectAutoscaling(ctx context.Context, projectID, branchID, endpointID string, retryConfig libpsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	endpoint, err := resolveEndpoint(ctx, w, projectID, branchID, endpointID)
	if err != nil {
		return err
	}

	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	if endpoint.Status == nil {
		return errors.New("endpoint status is not available")
	}

	if endpoint.Status.Hosts == nil || endpoint.Status.Hosts.Host == "" {
		return errors.New("endpoint host information is not available")
	}

	token, err := target.AutoscalingCredential(ctx, w, endpoint.Name)
	if err != nil {
		return err
	}

	var endpointType string
	switch endpoint.Status.EndpointType {
	case postgres.EndpointTypeEndpointTypeReadWrite:
		endpointType = "read-write"
	case postgres.EndpointTypeEndpointTypeReadOnly:
		endpointType = "read-only"
	default:
		endpointType = string(endpoint.Status.EndpointType)
	}

	state := endpoint.Status.CurrentState
	var suffix string
	switch state {
	case postgres.EndpointStatusStateActive:
		// No need to inform the user that the endpoint is active.
	case postgres.EndpointStatusStateIdle:
		suffix = " (idle, waking up)"
	default:
		return fmt.Errorf("endpoint is not ready for accepting connections (state: %s)", state)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Connecting to %s endpoint%s...", endpointType, suffix))

	return libpsql.Connect(ctx, libpsql.ConnectOptions{
		Host:            endpoint.Status.Hosts.Host,
		Username:        user.UserName,
		Password:        token,
		DefaultDatabase: autoscalingDefaultDatabase,
		ExtraArgs:       extraArgs,
	}, retryConfig)
}

// resolveEndpoint resolves a partial specification to a full endpoint.
// Uses interactive selection when components are missing.
func resolveEndpoint(ctx context.Context, w *databricks.WorkspaceClient, projectID, branchID, endpointID string) (*postgres.Endpoint, error) {
	// If project not specified, select one
	if projectID == "" {
		var err error
		projectID, err = selectProjectID(ctx, w)
		if err != nil {
			return nil, fmt.Errorf("failed to select project: %w", err)
		}
	}

	// Get project to display its name
	project, err := target.GetProject(ctx, w, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	projectDisplayName := projectID
	if project.Status != nil && project.Status.DisplayName != "" {
		projectDisplayName = project.Status.DisplayName
	}
	cmdio.LogString(ctx, "Project: "+projectDisplayName)

	// If branch not specified, select one
	if branchID == "" {
		branchID, err = selectBranchID(ctx, w, project.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to select branch: %w", err)
		}
	}

	// Get branch to validate it exists
	branch, err := w.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: project.Name + "/branches/" + branchID})
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	cmdio.LogString(ctx, "Branch: "+branchID)

	// If endpoint not specified, select one
	if endpointID == "" {
		endpointID, err = selectEndpointID(ctx, w, branch.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to select endpoint: %w", err)
		}
	}

	// Get endpoint to validate and return it
	endpoint, err := target.GetEndpoint(ctx, w, projectID, branchID, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}
	cmdio.LogString(ctx, "Endpoint: "+endpointID)

	return endpoint, nil
}

// selectAmbiguous prompts the user to pick one of the choices in an
// AmbiguousError. Caller is expected to have logged a header (e.g. via the
// spinner) before invoking. Used to keep psql's interactive UX while letting
// the shared lib do the actual list+filter work.
func selectAmbiguous(ctx context.Context, amb *target.AmbiguousError, prompt string) (string, error) {
	items := make([]cmdio.Tuple, 0, len(amb.Choices))
	for _, c := range amb.Choices {
		items = append(items, cmdio.Tuple{Name: c.DisplayName, Id: c.ID})
	}
	return cmdio.SelectOrdered(ctx, items, prompt)
}

// selectProjectID auto-selects if there's only one project, otherwise prompts user to select.
// Returns the project ID (not the full project object).
func selectProjectID(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading projects...")
	id, err := target.AutoSelectProject(ctx, w)
	sp.Close()

	var amb *target.AmbiguousError
	if !errors.As(err, &amb) {
		return id, err
	}
	return selectAmbiguous(ctx, amb, "Select project")
}

// selectBranchID auto-selects if there's only one branch, otherwise prompts user to select.
// Returns the branch ID (not the full branch object).
func selectBranchID(ctx context.Context, w *databricks.WorkspaceClient, projectName string) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading branches...")
	id, err := target.AutoSelectBranch(ctx, w, projectName)
	sp.Close()

	var amb *target.AmbiguousError
	if !errors.As(err, &amb) {
		return id, err
	}
	return selectAmbiguous(ctx, amb, "Select branch")
}

// selectEndpointID auto-selects if there's only one endpoint, otherwise prompts user to select.
// Returns the endpoint ID (not the full endpoint object).
func selectEndpointID(ctx context.Context, w *databricks.WorkspaceClient, branchName string) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading endpoints...")
	id, err := target.AutoSelectEndpoint(ctx, w, branchName)
	sp.Close()

	var amb *target.AmbiguousError
	if !errors.As(err, &amb) {
		return id, err
	}
	return selectAmbiguous(ctx, amb, "Select endpoint")
}
