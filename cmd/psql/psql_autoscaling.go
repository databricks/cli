package psql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	libpsql "github.com/databricks/cli/libs/psql"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// autoscalingDefaultDatabase is the default database for Lakebase Autoscaling projects.
const autoscalingDefaultDatabase = "databricks_postgres"

// extractIDFromName extracts the ID component from a resource name.
// For example, extractIDFromName("projects/foo/branches/bar", "branches") returns "bar".
func extractIDFromName(name, component string) string {
	parts := strings.Split(name, "/")
	for i := range len(parts) - 1 {
		if parts[i] == component {
			return parts[i+1]
		}
	}
	return name
}

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

	cred, err := w.Postgres.GenerateDatabaseCredential(ctx, postgres.GenerateDatabaseCredentialRequest{
		Endpoint: endpoint.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to get database credentials: %w", err)
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
		return errors.New("endpoint is not ready for accepting connections")
	}

	cmdio.LogString(ctx, fmt.Sprintf("Connecting to %s endpoint%s...", endpointType, suffix))

	return libpsql.Connect(ctx, libpsql.ConnectOptions{
		Host:            endpoint.Status.Hosts.Host,
		Username:        user.UserName,
		Password:        cred.Token,
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
	project, err := w.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: "projects/" + projectID})
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
	endpoint, err := w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: branch.Name + "/endpoints/" + endpointID})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}
	cmdio.LogString(ctx, "Endpoint: "+endpointID)

	return endpoint, nil
}

// selectProjectID auto-selects if there's only one project, otherwise prompts user to select.
// Returns the project ID (not the full project object).
func selectProjectID(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading projects...")
	projects, err := w.Postgres.ListProjectsAll(ctx, postgres.ListProjectsRequest{})
	sp.Close()
	if err != nil {
		return "", err
	}

	if len(projects) == 0 {
		return "", errors.New("no Lakebase Autoscaling projects found in workspace")
	}

	// Auto-select if there's only one project
	if len(projects) == 1 {
		return extractIDFromName(projects[0].Name, "projects"), nil
	}

	// Multiple projects, prompt user to select
	var items []cmdio.Tuple
	for _, project := range projects {
		projectID := extractIDFromName(project.Name, "projects")
		displayName := projectID
		if project.Status != nil && project.Status.DisplayName != "" {
			displayName = project.Status.DisplayName
		}
		items = append(items, cmdio.Tuple{Name: displayName, Id: projectID})
	}

	return cmdio.SelectOrdered(ctx, items, "Select project")
}

// selectBranchID auto-selects if there's only one branch, otherwise prompts user to select.
// Returns the branch ID (not the full branch object).
func selectBranchID(ctx context.Context, w *databricks.WorkspaceClient, projectName string) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading branches...")
	branches, err := w.Postgres.ListBranchesAll(ctx, postgres.ListBranchesRequest{
		Parent: projectName,
	})
	sp.Close()
	if err != nil {
		return "", err
	}

	if len(branches) == 0 {
		return "", errors.New("no branches found in project")
	}

	// Auto-select if there's only one branch
	if len(branches) == 1 {
		return extractIDFromName(branches[0].Name, "branches"), nil
	}

	// Multiple branches, prompt user to select
	var items []cmdio.Tuple
	for _, branch := range branches {
		branchID := extractIDFromName(branch.Name, "branches")
		items = append(items, cmdio.Tuple{Name: branchID, Id: branchID})
	}

	return cmdio.SelectOrdered(ctx, items, "Select branch")
}

// selectEndpointID auto-selects if there's only one endpoint, otherwise prompts user to select.
// Returns the endpoint ID (not the full endpoint object).
func selectEndpointID(ctx context.Context, w *databricks.WorkspaceClient, branchName string) (string, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading endpoints...")
	endpoints, err := w.Postgres.ListEndpointsAll(ctx, postgres.ListEndpointsRequest{
		Parent: branchName,
	})
	sp.Close()
	if err != nil {
		return "", err
	}

	if len(endpoints) == 0 {
		return "", errors.New("no endpoints found in branch")
	}

	// Auto-select if there's only one endpoint
	if len(endpoints) == 1 {
		return extractIDFromName(endpoints[0].Name, "endpoints"), nil
	}

	// Multiple endpoints, prompt user to select
	var items []cmdio.Tuple
	for _, endpoint := range endpoints {
		endpointID := extractIDFromName(endpoint.Name, "endpoints")
		items = append(items, cmdio.Tuple{Name: endpointID, Id: endpointID})
	}

	return cmdio.SelectOrdered(ctx, items, "Select endpoint")
}
