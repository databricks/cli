package dresources

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

func init() {
	registerTestDeps("postgres_snapshot_schedules", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		// Creating the project implicitly provisions the root "production"
		// branch, the only branch a snapshot schedule may target.
		_, err := client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
			ProjectId: "test-project-for-snapshot-schedule",
			Project: postgres.Project{
				Spec: &postgres.ProjectSpec{
					DisplayName: "Test Project for Snapshot Schedule",
					PgVersion:   16,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return &resources.PostgresSnapshotSchedule{
			Branch: "projects/test-project-for-snapshot-schedule/branches/production",
		}, nil
	})
}

func TestSnapshotScheduleName(t *testing.T) {
	const branch = "projects/p/branches/b"
	want := branch + "/snapshot-schedule"

	cases := []string{
		branch,
		branch + "/",
		branch + "//",
	}
	for _, in := range cases {
		if got := snapshotScheduleName(in); got != want {
			t.Errorf("snapshotScheduleName(%q) = %q, want %q", in, got, want)
		}
	}
}
