package psql

import (
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestParseResourcePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		project  string
		branch   string
		endpoint string
		wantErr  string
	}{
		{
			name:    "project only",
			input:   "projects/my-project",
			project: "my-project",
		},
		{
			name:    "project and branch",
			input:   "projects/my-project/branches/main",
			project: "my-project",
			branch:  "main",
		},
		{
			name:     "full path",
			input:    "projects/my-project/branches/main/endpoints/primary",
			project:  "my-project",
			branch:   "main",
			endpoint: "primary",
		},
		{
			name:    "missing project ID",
			input:   "projects/",
			wantErr: "missing project ID",
		},
		{
			name:    "missing branch ID",
			input:   "projects/my-project/branches/",
			wantErr: "missing branch ID",
		},
		{
			name:    "missing endpoint ID",
			input:   "projects/my-project/branches/main/endpoints/",
			wantErr: "missing endpoint ID",
		},
		{
			name:    "invalid segment after project",
			input:   "projects/my-project/invalid/foo",
			wantErr: "expected 'branches' after project",
		},
		{
			name:    "invalid segment after branch",
			input:   "projects/my-project/branches/main/invalid/foo",
			wantErr: "expected 'endpoints' after branch",
		},
		{
			name:    "not a projects path",
			input:   "something/else",
			wantErr: "invalid resource path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			project, branch, endpoint, err := parseResourcePath(tc.input)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.project, project)
			assert.Equal(t, tc.branch, branch)
			assert.Equal(t, tc.endpoint, endpoint)
		})
	}
}

func TestListAllDatabases(t *testing.T) {
	instErr := errors.New("instances list failed")
	projErr := errors.New("projects list failed")
	instances := []database.DatabaseInstance{{Name: "my-instance"}}
	projects := []postgres.Project{{Name: "projects/my-project"}}

	tests := []struct {
		name          string
		instErr       error
		projErr       error
		wantInstances []database.DatabaseInstance
		wantProjects  []postgres.Project
		wantErr       bool
	}{
		{
			name:          "both succeed",
			wantInstances: instances,
			wantProjects:  projects,
		},
		{
			name:         "instances call fails",
			instErr:      instErr,
			wantProjects: projects,
		},
		{
			name:          "projects call fails",
			projErr:       projErr,
			wantInstances: instances,
		},
		{
			name:    "both calls fail",
			instErr: instErr,
			projErr: projErr,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := mocks.NewMockWorkspaceClient(t)

			var instReturn []database.DatabaseInstance
			if tc.instErr == nil {
				instReturn = instances
			}
			m.GetMockDatabaseAPI().EXPECT().
				ListDatabaseInstancesAll(mock.Anything, database.ListDatabaseInstancesRequest{}).
				Return(instReturn, tc.instErr)

			var projReturn []postgres.Project
			if tc.projErr == nil {
				projReturn = projects
			}
			m.GetMockPostgresAPI().EXPECT().
				ListProjectsAll(mock.Anything, postgres.ListProjectsRequest{}).
				Return(projReturn, tc.projErr)

			gotInstances, gotProjects, err := listAllDatabases(t.Context(), m.WorkspaceClient)
			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.instErr)
				assert.ErrorIs(t, err, tc.projErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantInstances, gotInstances)
			assert.Equal(t, tc.wantProjects, gotProjects)
		})
	}
}
