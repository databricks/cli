package target

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/listing"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	postgres "github.com/databricks/sdk-go/postgres/v1"
)

// ListProjects returns all autoscaling projects in the workspace.
func ListProjects(ctx context.Context, w *sdkclients.Clients) ([]postgres.Project, error) {
	return listing.Collect(w.Postgres.ListProjectsIter(ctx, postgres.ListProjectsRequest{}))
}

// ListBranches returns all branches under the given project.
// projectName is the SDK resource name like "projects/foo".
func ListBranches(ctx context.Context, w *sdkclients.Clients, projectName string) ([]postgres.Branch, error) {
	return listing.Collect(w.Postgres.ListBranchesIter(ctx, postgres.ListBranchesRequest{Parent: new(projectName)}))
}

// ListEndpoints returns all endpoints under the given branch.
// branchName is the SDK resource name like "projects/foo/branches/bar".
func ListEndpoints(ctx context.Context, w *sdkclients.Clients, branchName string) ([]postgres.Endpoint, error) {
	return listing.Collect(w.Postgres.ListEndpointsIter(ctx, postgres.ListEndpointsRequest{Parent: new(branchName)}))
}

// GetProject fetches a single project by ID. Unlike GetProvisioned, the
// Postgres autoscaling API populates the Name field on the response so we do
// not need to patch it.
func GetProject(ctx context.Context, w *sdkclients.Clients, projectID string) (*postgres.Project, error) {
	return w.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: new(pathSegmentProjects + "/" + projectID)})
}

// GetEndpoint fetches a single endpoint by ID, given its parent IDs. Unlike
// GetProvisioned, the Postgres autoscaling API populates the Name field.
func GetEndpoint(ctx context.Context, w *sdkclients.Clients, projectID, branchID, endpointID string) (*postgres.Endpoint, error) {
	name := fmt.Sprintf("projects/%s/branches/%s/endpoints/%s", projectID, branchID, endpointID)
	return w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: new(name)})
}

// AutoSelectProject returns the trailing project ID (e.g. "foo", not
// "projects/foo") if exactly one project exists. Returns an *AmbiguousError
// carrying the choices if there are multiple, or a plain error if there are none.
func AutoSelectProject(ctx context.Context, w *sdkclients.Clients) (string, error) {
	projects, err := ListProjects(ctx, w)
	if err != nil {
		return "", err
	}
	if len(projects) == 0 {
		return "", errors.New("no Lakebase Autoscaling projects found in workspace")
	}
	if len(projects) == 1 {
		return extractID(utils.Deref(projects[0].Name), pathSegmentProjects), nil
	}

	choices := make([]Choice, 0, len(projects))
	for _, p := range projects {
		id := extractID(utils.Deref(p.Name), pathSegmentProjects)
		var display string
		if p.Status != nil && utils.Deref(p.Status.DisplayName) != "" && utils.Deref(p.Status.DisplayName) != id {
			display = utils.Deref(p.Status.DisplayName)
		}
		choices = append(choices, Choice{ID: id, DisplayName: display})
	}
	return "", &AmbiguousError{Kind: KindProject, FlagHint: "--project", Choices: choices}
}

// AutoSelectBranch returns the trailing branch ID under projectName if
// exactly one branch exists. Returns an *AmbiguousError if there are multiple.
// projectName is the SDK resource name (e.g. "projects/foo").
func AutoSelectBranch(ctx context.Context, w *sdkclients.Clients, projectName string) (string, error) {
	branches, err := ListBranches(ctx, w, projectName)
	if err != nil {
		return "", err
	}
	if len(branches) == 0 {
		return "", errors.New("no branches found in project")
	}
	if len(branches) == 1 {
		return extractID(utils.Deref(branches[0].Name), pathSegmentBranches), nil
	}

	choices := make([]Choice, 0, len(branches))
	for _, b := range branches {
		id := extractID(utils.Deref(b.Name), pathSegmentBranches)
		choices = append(choices, Choice{ID: id})
	}
	return "", &AmbiguousError{Kind: KindBranch, Parent: projectName, FlagHint: "--branch", Choices: choices}
}

// AutoSelectEndpoint returns the trailing endpoint ID under branchName if
// exactly one endpoint exists. Returns an *AmbiguousError if there are multiple.
// branchName is the SDK resource name (e.g. "projects/foo/branches/bar").
func AutoSelectEndpoint(ctx context.Context, w *sdkclients.Clients, branchName string) (string, error) {
	endpoints, err := ListEndpoints(ctx, w, branchName)
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		return "", errors.New("no endpoints found in branch")
	}
	if len(endpoints) == 1 {
		return extractID(utils.Deref(endpoints[0].Name), pathSegmentEndpoints), nil
	}

	choices := make([]Choice, 0, len(endpoints))
	for _, e := range endpoints {
		id := extractID(utils.Deref(e.Name), pathSegmentEndpoints)
		choices = append(choices, Choice{ID: id})
	}
	return "", &AmbiguousError{Kind: KindEndpoint, Parent: branchName, FlagHint: "--endpoint", Choices: choices}
}

// AutoscalingCredential issues a short-lived OAuth token that can be used to
// authenticate to the given autoscaling endpoint. endpointName is the SDK
// resource name (e.g. "projects/foo/branches/bar/endpoints/baz").
func AutoscalingCredential(ctx context.Context, w *sdkclients.Clients, endpointName string) (string, error) {
	cred, err := w.Postgres.GenerateDatabaseCredential(ctx, postgres.GenerateDatabaseCredentialRequest{
		Endpoint: new(endpointName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get database credentials: %w", err)
	}
	return utils.Deref(cred.Token), nil
}
