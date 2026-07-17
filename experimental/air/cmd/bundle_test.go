package aircmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAirJob(t *testing.T) {
	airJob := jobs.BaseJob{Settings: &jobs.JobSettings{
		Tasks: []jobs.Task{{AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"}}},
	}}
	assert.True(t, isAirJob(airJob))

	sweep := jobs.BaseJob{Settings: &jobs.JobSettings{
		Tasks: []jobs.Task{{ForEachTask: &jobs.ForEachTask{
			Task: jobs.Task{AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp"}},
		}}},
	}}
	assert.True(t, isAirJob(sweep), "foreach-wrapped ai_runtime_task is an AIR job")

	legacy := jobs.BaseJob{Settings: &jobs.JobSettings{
		Tasks: []jobs.Task{{GenAiComputeTask: &jobs.GenAiComputeTask{TrainingScriptPath: "/x"}}},
	}}
	assert.True(t, isAirJob(legacy))

	notebook := jobs.BaseJob{Settings: &jobs.JobSettings{
		Tasks: []jobs.Task{{NotebookTask: &jobs.NotebookTask{NotebookPath: "/x"}}},
	}}
	assert.False(t, isAirJob(notebook))

	assert.False(t, isAirJob(jobs.BaseJob{}), "a job with no settings is not an AIR job")
}

// jobsListServer serves one jobs/list page and stubs everything else (config
// probe, scim Me). The Me response fixes the current user for scoping tests.
func jobsListServer(t *testing.T, user string, jobList ...jobs.BaseJob) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(jobs.ListJobsResponse{Jobs: jobList})
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.2/jobs/list":
			_, _ = w.Write(body)
		case "/api/2.0/preview/scim/v2/Me":
			_, _ = w.Write([]byte(`{"userName":"` + user + `"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func airBaseJob(jobID int64, name, user string) jobs.BaseJob {
	return jobs.BaseJob{
		JobId:           jobID,
		CreatorUserName: user,
		Settings: &jobs.JobSettings{
			Name:  name,
			Tasks: []jobs.Task{{AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: name}}},
		},
	}
}

func TestListAirBundlesFiltersNonAirAndUser(t *testing.T) {
	jobList := []jobs.BaseJob{
		airBaseJob(1, "exp-a", "me@example.com"),
		{JobId: 2, CreatorUserName: "me@example.com", Settings: &jobs.JobSettings{ // not AIR
			Name: "nb", Tasks: []jobs.Task{{NotebookTask: &jobs.NotebookTask{NotebookPath: "/x"}}},
		}},
		airBaseJob(3, "exp-b", "other@example.com"), // wrong user
		airBaseJob(5, "exp-c", "me@example.com"),
	}
	srv := jobsListServer(t, "me@example.com", jobList...)
	w := newTestWorkspaceClient(t, srv.URL)

	bundles, truncated, err := listAirBundles(t.Context(), w, "me@example.com")
	require.NoError(t, err)
	assert.False(t, truncated)
	require.Len(t, bundles, 2)
	assert.Equal(t, "exp-a", bundles[0].Name)
	assert.Equal(t, "exp-c", bundles[1].Name)

	// No user filter keeps every AIR job (still drops the non-AIR notebook job).
	all, _, err := listAirBundles(t.Context(), w, "")
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestResolveBundleNotFound(t *testing.T) {
	srv := jobsListServer(t, "me@example.com", airBaseJob(1, "exp-a", "me@example.com"))
	w := newTestWorkspaceClient(t, srv.URL)

	_, err := resolveBundle(t.Context(), w, "does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `bundle "does-not-exist" not found`)

	got, err := resolveBundle(t.Context(), w, "exp-a")
	require.NoError(t, err)
	assert.Equal(t, "1", got.JobID)
}

func TestDeleteBundleArgs(t *testing.T) {
	tests := []struct {
		name    string
		all     bool
		args    []string
		wantErr string
	}{
		{name: "one name", args: []string{"exp-a"}},
		{name: "many names", args: []string{"exp-a", "exp-b"}},
		{name: "all", all: true},
		{name: "no input", wantErr: "provide at least one bundle NAME, or use --all"},
		{name: "names with all", all: true, args: []string{"exp-a"}, wantErr: "cannot combine NAME arguments with --all"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newDeleteBundleCommand()
			if tc.all {
				require.NoError(t, cmd.Flags().Set("all", "true"))
			}
			err := cmd.Args(cmd, tc.args)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestDeleteTargetsResolvesNames(t *testing.T) {
	srv := jobsListServer(t, "me@example.com",
		airBaseJob(1, "exp-a", "me@example.com"),
		airBaseJob(2, "exp-b", "me@example.com"),
	)
	w := newTestWorkspaceClient(t, srv.URL)

	// --all returns every bundle.
	all, err := deleteTargets(t.Context(), w, nil, true)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// Named targets resolve to their bundles; an unknown name errors.
	named, err := deleteTargets(t.Context(), w, []string{"exp-b"}, false)
	require.NoError(t, err)
	require.Len(t, named, 1)
	assert.Equal(t, "2", named[0].JobID)

	_, err = deleteTargets(t.Context(), w, []string{"nope"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `bundle "nope" not found`)
}
