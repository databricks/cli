package aircmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDlRuntimeImage(t *testing.T) {
	ctx := t.Context()
	// A config runtime version wins and is used bare.
	assert.Equal(t, "5", dlRuntimeImage(ctx, "5"))
	// The CLIENT-GPU- prefix is always stripped, even from the config version.
	assert.Equal(t, "5", dlRuntimeImage(ctx, "CLIENT-GPU-5"))
	// Default, with the prefix stripped.
	assert.Equal(t, "4", dlRuntimeImage(ctx, ""))
	// Env override, prefix stripped.
	t.Setenv(dlRuntimeImageEnv, "CLIENT-GPU-7")
	assert.Equal(t, "7", dlRuntimeImage(ctx, ""))
}

func TestBuildSubmitPayload(t *testing.T) {
	cfg := &runConfig{
		ExperimentName:            "exp",
		Command:                   new("python train.py"),
		Compute:                   &computeConfig{AcceleratorType: "GPU_8xH100", NumAccelerators: 16},
		MaxRetries:                new(2),
		TimeoutMinutes:            new(30),
		MLflowRunName:             new("run-v2"),
		MLflowExperimentDirectory: new("/Workspace/Users/me/exp"),
	}

	p := buildSubmitPayload(cfg, "/d/command.sh", "5", snapshotResult{})

	assert.Equal(t, "exp", p.RunName)
	assert.Equal(t, 1800, p.TimeoutSeconds)
	require.Len(t, p.Environments, 1)
	assert.Equal(t, aiRuntimeEnvironmentKey, p.Environments[0].EnvironmentKey)
	require.NotNil(t, p.Environments[0].Spec)
	assert.Equal(t, "5", p.Environments[0].Spec.EnvironmentVersion)

	require.Len(t, p.Tasks, 1)
	task := p.Tasks[0]
	assert.Equal(t, "exp", task.TaskKey)
	assert.Equal(t, jobs.RunIfAllSuccess, task.RunIf)
	assert.Equal(t, aiRuntimeEnvironmentKey, task.EnvironmentKey)
	assert.Equal(t, 2, task.MaxRetries)
	assert.True(t, task.RetryOnTimeout)

	at := task.AiRuntimeTask
	require.NotNil(t, at)
	assert.Equal(t, "exp", at.Experiment)
	assert.Equal(t, "run-v2", at.MlflowRun)
	assert.Equal(t, "/Workspace/Users/me/exp", at.MlflowExperimentDirectory)
	require.Len(t, at.Deployments, 1)
	assert.Equal(t, "/d/command.sh", at.Deployments[0].CommandPath)
	assert.Equal(t, jobs.ComputeSpec{AcceleratorType: jobs.ComputeSpecAcceleratorTypeGpu8xH100, AcceleratorCount: 16}, at.Deployments[0].Compute)
}

func TestBuildSubmitPayloadDefaultRetries(t *testing.T) {
	// max_retries unset defaults to 3 (matching the Python native path), so both
	// retry fields are sent.
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("x"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xH100", NumAccelerators: 1},
	}
	task := buildSubmitPayload(cfg, "/d/command.sh", "4", snapshotResult{}).Tasks[0]
	assert.Equal(t, defaultMaxRetries, task.MaxRetries)
	assert.True(t, task.RetryOnTimeout)
}

func TestBuildSubmitPayloadNoRetries(t *testing.T) {
	// max_retries: 0 must be sent explicitly so Jobs honors "no retries" instead
	// of applying the server default. retry_on_timeout is omitted when retries
	// aren't allowed.
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("x"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xH100", NumAccelerators: 1},
		MaxRetries:     new(0),
	}
	task := buildSubmitPayload(cfg, "/d/command.sh", "4", snapshotResult{}).Tasks[0]
	assert.Equal(t, 0, task.MaxRetries)
	assert.False(t, task.RetryOnTimeout)

	b, err := json.Marshal(task)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"max_retries":0`)
	assert.NotContains(t, string(b), "retry_on_timeout")
}

func TestSubmitToken(t *testing.T) {
	cfg := &runConfig{IdempotencyToken: new("from-config")}

	tok, err := submitToken("from-flag", cfg) // flag wins
	require.NoError(t, err)
	assert.Equal(t, "from-flag", tok)

	tok, err = submitToken("", cfg) // then config
	require.NoError(t, err)
	assert.Equal(t, "from-config", tok)

	tok, err = submitToken("", &runConfig{}) // else generated
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	// An over-long token errors instead of being truncated.
	_, err = submitToken(strings.Repeat("a", 65), cfg)
	require.ErrorContains(t, err, "64 characters or less")
}

func TestSubmitWorkload(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	// Register before AddDefaultHandlers: the router is first-wins, so this must claim the route ahead of the default handler.
	var got jobs.SubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return jobs.SubmitRunResponse{RunId: 777}
	})
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cfg, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	runID, dashboardURL, err := submitWorkload(t.Context(), w, cfg, cfgPath, "idem-key")
	require.NoError(t, err)
	assert.Equal(t, int64(777), runID)
	assert.Contains(t, dashboardURL, "/jobs/runs/777")

	// The submitted payload is a native ai_runtime_task pointing at the uploaded
	// command.sh under the run's launch directory.
	assert.Equal(t, "my-run", got.RunName)
	assert.Equal(t, "idem-key", got.IdempotencyToken)
	require.Len(t, got.Environments, 1)
	require.Len(t, got.Tasks, 1)
	at := got.Tasks[0].AiRuntimeTask
	require.NotNil(t, at)
	require.Len(t, at.Deployments, 1)
	d := at.Deployments[0]
	assert.True(t, strings.HasSuffix(d.CommandPath, "/"+commandScriptName), d.CommandPath)
	assert.Contains(t, d.CommandPath, "/.air/cli_launch/")
	assert.Equal(t, jobs.ComputeSpec{AcceleratorType: jobs.ComputeSpecAcceleratorTypeGpu1xH100, AcceleratorCount: 1}, d.Compute)
}

// TestSubmitWorkloadWithCodeSource exercises the snapshot path end to end: a
// git-pinned code_source is packaged, uploaded, and its paths attached to the task.
func TestSubmitWorkloadWithCodeSource(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	// Register before AddDefaultHandlers: the router is first-wins, so this must claim the route ahead of the default handler.
	var got jobs.SubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return jobs.SubmitRunResponse{RunId: 555}
	})
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	// A git repo committed at HEAD, referenced by commit so packaging is git_archive.
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print()")
	sha := commitAll(t, repo, "init")

	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ` + repo + `
    git:
      commit: ` + sha + `
`
	cfgPath := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	_, _, err = submitWorkload(t.Context(), w, loaded, cfgPath, "idem")
	require.NoError(t, err)

	at := got.Tasks[0].AiRuntimeTask
	// The tarball path is under the user's repo_snapshots dir. git_state_path /
	// git_diff_path are not asserted: the typed jobs.AiRuntimeTask has no such fields
	// (see the TEMP note in buildSubmitPayload), so they aren't sent. The git_state
	// sidecar file is still uploaded next to the tarball — covered by TestRunSnapshot.
	assert.Contains(t, at.CodeSourcePath, "/.air/repo_snapshots/"+filepath.Base(repo)+"/")
	assert.True(t, strings.HasSuffix(at.CodeSourcePath, ".tar.gz"), at.CodeSourcePath)
}

func TestSubmitWorkloadGuards(t *testing.T) {
	w := newFakeWorkspaceClient(t)
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	base, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	t.Run("usage_policy_name rejected", func(t *testing.T) {
		cfg := *base
		cfg.UsagePolicyName = new("p")
		_, _, err := submitWorkload(t.Context(), w, &cfg, cfgPath, "")
		require.ErrorContains(t, err, "usage_policy_name is not yet supported")
	})
}
