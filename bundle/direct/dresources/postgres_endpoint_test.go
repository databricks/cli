package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

func init() {
	registerTestDeps("postgres_endpoints", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		// Create parent project first
		_, err := client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
			ProjectId: "test-project-for-endpoint",
			Project: postgres.Project{
				Spec: &postgres.ProjectSpec{
					DisplayName: "Test Project for Endpoint",
					PgVersion:   16,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		// Create parent branch
		_, err = client.Postgres.CreateBranch(ctx, postgres.CreateBranchRequest{
			Parent:   "projects/test-project-for-endpoint",
			BranchId: "test-branch-for-endpoint",
			Branch:   postgres.Branch{},
		})
		if err != nil {
			return nil, err
		}

		return &resources.PostgresEndpoint{
			Parent:       "projects/test-project-for-endpoint/branches/test-branch-for-endpoint",
			EndpointId:   "test-endpoint",
			EndpointType: postgres.EndpointTypeEndpointTypeReadWrite,
		}, nil
	})

	registerKnownMissingInRemoteType("postgres_endpoints",
		"replace_existing",
	)
}
