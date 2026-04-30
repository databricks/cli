package target

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// ListProjects returns all autoscaling projects in the workspace.
func ListProjects(ctx context.Context, w *databricks.WorkspaceClient) ([]postgres.Project, error) {
	return w.Postgres.ListProjectsAll(ctx, postgres.ListProjectsRequest{})
}

// ListBranches returns all branches under the given project.
// projectName is the SDK resource name like "projects/foo".
func ListBranches(ctx context.Context, w *databricks.WorkspaceClient, projectName string) ([]postgres.Branch, error) {
	return w.Postgres.ListBranchesAll(ctx, postgres.ListBranchesRequest{Parent: projectName})
}

// ListEndpoints returns all endpoints under the given branch.
// branchName is the SDK resource name like "projects/foo/branches/bar".
func ListEndpoints(ctx context.Context, w *databricks.WorkspaceClient, branchName string) ([]postgres.Endpoint, error) {
	return w.Postgres.ListEndpointsAll(ctx, postgres.ListEndpointsRequest{Parent: branchName})
}

// GetProject fetches a single project by ID.
func GetProject(ctx context.Context, w *databricks.WorkspaceClient, projectID string) (*postgres.Project, error) {
	return w.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: PathSegmentProjects + "/" + projectID})
}

// GetEndpoint fetches a single endpoint by ID, given its parent IDs.
func GetEndpoint(ctx context.Context, w *databricks.WorkspaceClient, projectID, branchID, endpointID string) (*postgres.Endpoint, error) {
	name := fmt.Sprintf("projects/%s/branches/%s/endpoints/%s", projectID, branchID, endpointID)
	return w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: name})
}

// AutoSelectProject returns the only project in the workspace, or an
// AmbiguousError carrying the choices if there are multiple. Returns a plain
// error if there are no projects.
func AutoSelectProject(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	projects, err := ListProjects(ctx, w)
	if err != nil {
		return "", err
	}
	if len(projects) == 0 {
		return "", errors.New("no Lakebase Autoscaling projects found in workspace")
	}
	if len(projects) == 1 {
		return ExtractID(projects[0].Name, PathSegmentProjects), nil
	}

	choices := make([]Choice, 0, len(projects))
	for _, p := range projects {
		id := ExtractID(p.Name, PathSegmentProjects)
		display := id
		if p.Status != nil && p.Status.DisplayName != "" {
			display = p.Status.DisplayName
		}
		choices = append(choices, Choice{ID: id, DisplayName: display})
	}
	return "", &AmbiguousError{Kind: "project", FlagHint: "--project", Choices: choices}
}

// AutoSelectBranch returns the only branch under projectName, or an
// AmbiguousError if there are multiple.
func AutoSelectBranch(ctx context.Context, w *databricks.WorkspaceClient, projectName string) (string, error) {
	branches, err := ListBranches(ctx, w, projectName)
	if err != nil {
		return "", err
	}
	if len(branches) == 0 {
		return "", errors.New("no branches found in project")
	}
	if len(branches) == 1 {
		return ExtractID(branches[0].Name, pathSegmentBranches), nil
	}

	choices := make([]Choice, 0, len(branches))
	for _, b := range branches {
		id := ExtractID(b.Name, pathSegmentBranches)
		choices = append(choices, Choice{ID: id, DisplayName: id})
	}
	return "", &AmbiguousError{Kind: "branch", Parent: projectName, FlagHint: "--branch", Choices: choices}
}

// AutoSelectEndpoint returns the only endpoint under branchName, or an
// AmbiguousError if there are multiple.
func AutoSelectEndpoint(ctx context.Context, w *databricks.WorkspaceClient, branchName string) (string, error) {
	endpoints, err := ListEndpoints(ctx, w, branchName)
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		return "", errors.New("no endpoints found in branch")
	}
	if len(endpoints) == 1 {
		return ExtractID(endpoints[0].Name, pathSegmentEndpoints), nil
	}

	choices := make([]Choice, 0, len(endpoints))
	for _, e := range endpoints {
		id := ExtractID(e.Name, pathSegmentEndpoints)
		choices = append(choices, Choice{ID: id, DisplayName: id})
	}
	return "", &AmbiguousError{Kind: "endpoint", Parent: branchName, FlagHint: "--endpoint", Choices: choices}
}

// AutoscalingCredential issues a short-lived OAuth token that can be used to
// authenticate to the given autoscaling endpoint. endpointName is the SDK
// resource name (e.g. "projects/foo/branches/bar/endpoints/baz").
func AutoscalingCredential(ctx context.Context, w *databricks.WorkspaceClient, endpointName string) (string, error) {
	cred, err := w.Postgres.GenerateDatabaseCredential(ctx, postgres.GenerateDatabaseCredentialRequest{
		Endpoint: endpointName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get database credentials: %w", err)
	}
	return cred.Token, nil
}
