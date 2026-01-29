package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	lakebasepsql "github.com/databricks/cli/libs/lakebase/psql"
	lakebasev2 "github.com/databricks/cli/libs/lakebase/v2"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// connectAutoscaling connects to a Lakebase Autoscaling endpoint.
func connectAutoscaling(ctx context.Context, projectID, branchID, endpointID string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	endpoint, err := resolveEndpoint(ctx, w, projectID, branchID, endpointID)
	if err != nil {
		return err
	}

	return lakebasev2.Connect(ctx, w, endpoint, retryConfig, extraArgs...)
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
		return lakebasev2.ExtractIDFromName(projects[0].Name, "projects"), nil
	}

	// Multiple projects, prompt user to select
	var items []cmdio.Tuple
	for _, project := range projects {
		projectID := lakebasev2.ExtractIDFromName(project.Name, "projects")
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
		return lakebasev2.ExtractIDFromName(branches[0].Name, "branches"), nil
	}

	// Multiple branches, prompt user to select
	var items []cmdio.Tuple
	for _, branch := range branches {
		branchID := lakebasev2.ExtractIDFromName(branch.Name, "branches")
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
		return lakebasev2.ExtractIDFromName(endpoints[0].Name, "endpoints"), nil
	}

	// Multiple endpoints, prompt user to select
	var items []cmdio.Tuple
	for _, endpoint := range endpoints {
		endpointID := lakebasev2.ExtractIDFromName(endpoint.Name, "endpoints")
		items = append(items, cmdio.Tuple{Name: endpointID, Id: endpointID})
	}

	return cmdio.SelectOrdered(ctx, items, "Select endpoint")
}
