package psql

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/databricks-sdk-go/config"
	database "github.com/databricks/sdk-go/database/v1"
	postgres "github.com/databricks/sdk-go/postgres/v1"
	"github.com/stretchr/testify/assert"
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
	const instancesBody = `{"database_instances":[{"name":"my-instance"}]}`
	const projectsBody = `{"projects":[{"name":"projects/my-project"}]}`
	instances := []database.DatabaseInstance{{Name: new("my-instance")}}
	projects := []postgres.Project{{Name: new("projects/my-project")}}

	tests := []struct {
		name          string
		instFails     bool
		projFails     bool
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
			instFails:    true,
			wantProjects: projects,
		},
		{
			name:          "projects call fails",
			projFails:     true,
			wantInstances: instances,
		},
		{
			name:      "both calls fail",
			instFails: true,
			projFails: true,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(fails bool, body, message string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if fails {
						w.WriteHeader(http.StatusInternalServerError)
						body = `{"error_code":"INTERNAL_ERROR","message":"` + message + `"}`
					}
					_, _ = w.Write([]byte(body))
				}
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/2.0/database/instances":
					handler(tc.instFails, instancesBody, "instances list failed")(w, r)
				case "/api/2.0/postgres/projects":
					handler(tc.projFails, projectsBody, "projects list failed")(w, r)
				default:
					_, _ = w.Write([]byte(`{}`))
				}
			}))
			t.Cleanup(srv.Close)
			w, err := sdkclients.New(t.Context(), &config.Config{Host: srv.URL, Token: "token"})
			require.NoError(t, err)

			gotInstances, gotProjects, err := listAllDatabases(t.Context(), w)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "instances list failed")
				assert.Contains(t, err.Error(), "projects list failed")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantInstances, gotInstances)
			assert.Equal(t, tc.wantProjects, gotProjects)
		})
	}
}
