package aircmd

import (
	"strconv"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scheduledConfig = minimalConfig + `
schedule:
  quartz_cron_expression: "0 0 9 * * ?"
  timezone_id: America/Los_Angeles
  pause_status: PAUSED
`

// buildJobSettings carries the same ai_runtime_task/environment as a submit, on a
// persistent Task with the cron schedule attached.
func TestBuildJobSettings(t *testing.T) {
	cfg, err := loadRunConfig(writeConfigFile(t, "run.yaml", scheduledConfig))
	require.NoError(t, err)

	js := buildJobSettings(cfg, "/d/command.sh", "5", "policy-1", snapshotResult{}, []string{"numpy"})
	assert.Equal(t, "my-run", js.Name)
	assert.Equal(t, "policy-1", js.BudgetPolicyId)

	require.Len(t, js.Tasks, 1)
	tk := js.Tasks[0]
	assert.Equal(t, "my-run", tk.TaskKey)
	assert.Equal(t, aiRuntimeEnvironmentKey, tk.EnvironmentKey)
	require.NotNil(t, tk.AiRuntimeTask)
	assert.Equal(t, "my-run", tk.AiRuntimeTask.Experiment)
	assert.Equal(t, "/d/command.sh", tk.AiRuntimeTask.Deployments[0].CommandPath)

	require.Len(t, js.Environments, 1)
	require.NotNil(t, js.Schedule)
	assert.Equal(t, "0 0 9 * * ?", js.Schedule.QuartzCronExpression)
	assert.Equal(t, "America/Los_Angeles", js.Schedule.TimezoneId)
	assert.Equal(t, jobs.PauseStatusPaused, js.Schedule.PauseStatus)
}

func TestBuildCronSchedule(t *testing.T) {
	assert.Nil(t, buildCronSchedule(nil))

	// An empty pause_status is left off so the Jobs default (UNPAUSED) applies.
	c := buildCronSchedule(&scheduleConfig{QuartzCronExpression: "* * * * * ?", TimezoneID: "UTC"})
	require.NotNil(t, c)
	assert.Equal(t, jobs.PauseStatus(""), c.PauseStatus)
}

func loadScheduledConfig(t *testing.T) (*databricks.WorkspaceClient, *runConfig, string) {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	cfgPath := writeConfigFile(t, "run.yaml", scheduledConfig)
	cfg, err := loadRunConfig(cfgPath)
	require.NoError(t, err)
	return w, cfg, cfgPath
}

// A scheduled run creates a persistent job the first time and updates it in place
// on a re-run (upsert by name), so re-running never piles up duplicate jobs.
func TestCreateScheduledJobCreatesThenUpdates(t *testing.T) {
	w, cfg, cfgPath := loadScheduledConfig(t)

	id1, url1, created, err := createScheduledJob(t.Context(), w, cfg, cfgPath)
	require.NoError(t, err)
	assert.True(t, created)
	assert.NotZero(t, id1)
	assert.Contains(t, url1, "/jobs/"+strconv.FormatInt(id1, 10))

	id2, _, created2, err := createScheduledJob(t.Context(), w, cfg, cfgPath)
	require.NoError(t, err)
	assert.False(t, created2, "a re-run with the same name updates in place")
	assert.Equal(t, id1, id2)

	// Exactly one job carries the name, and it kept the schedule through the reset.
	var found []jobs.BaseJob
	it := w.Jobs.List(t.Context(), jobs.ListJobsRequest{Name: cfg.ExperimentName})
	for it.HasNext(t.Context()) {
		j, err := it.Next(t.Context())
		require.NoError(t, err)
		if j.Settings != nil && j.Settings.Name == cfg.ExperimentName {
			found = append(found, j)
		}
	}
	require.Len(t, found, 1)
	require.NotNil(t, found[0].Settings.Schedule)
	assert.Equal(t, "0 0 9 * * ?", found[0].Settings.Schedule.QuartzCronExpression)
}

// When two jobs share the experiment name, the CLI can't tell which to update, so
// it errors rather than guessing.
func TestCreateScheduledJobAmbiguousName(t *testing.T) {
	w, cfg, cfgPath := loadScheduledConfig(t)

	for range 2 {
		_, err := w.Jobs.Create(t.Context(), jobs.CreateJob{Name: cfg.ExperimentName})
		require.NoError(t, err)
	}

	_, _, _, err := createScheduledJob(t.Context(), w, cfg, cfgPath)
	require.ErrorContains(t, err, "not unique")
}
