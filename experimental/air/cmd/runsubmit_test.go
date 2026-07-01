package aircmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
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

	p := buildSubmitPayload(cfg, "/d/command.sh", "5")

	assert.Equal(t, "exp", p.RunName)
	assert.Equal(t, 1800, p.TimeoutSeconds)
	require.Len(t, p.Environments, 1)
	assert.Equal(t, aiRuntimeEnvironmentKey, p.Environments[0].EnvironmentKey)
	assert.Equal(t, "5", p.Environments[0].Spec.EnvironmentVersion)

	require.Len(t, p.Tasks, 1)
	task := p.Tasks[0]
	assert.Equal(t, "exp", task.TaskKey)
	assert.Equal(t, "ALL_SUCCESS", task.RunIf)
	assert.Equal(t, aiRuntimeEnvironmentKey, task.EnvironmentKey)
	assert.Equal(t, 2, task.MaxRetries)
	assert.True(t, task.RetryOnTimeout)

	at := task.AiRuntimeTask
	assert.Equal(t, "exp", at.Experiment)
	assert.Equal(t, "run-v2", at.MlflowRun)
	assert.Equal(t, "/Workspace/Users/me/exp", at.MlflowExperimentDirectory)
	require.Len(t, at.Deployments, 1)
	assert.Equal(t, "/d/command.sh", at.Deployments[0].CommandPath)
	assert.Equal(t, aiRuntimeCompute{AcceleratorType: "GPU_8xH100", AcceleratorCount: 16}, at.Deployments[0].Compute)
}

func TestBuildSubmitPayload_NoRetries(t *testing.T) {
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("x"),
		Compute:        &computeConfig{AcceleratorType: "GPU_1xH100", NumAccelerators: 1},
		MaxRetries:     new(0),
	}

	task := buildSubmitPayload(cfg, "/d/command.sh", "4").Tasks[0]
	assert.Equal(t, 0, task.MaxRetries)
	assert.False(t, task.RetryOnTimeout)

	// max_retries: 0 must be sent, not omitted, so the server honors "no retries".
	b, err := json.Marshal(task)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"max_retries":0`)
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

func TestJobsSubmitClient(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var got jobsSubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return submitRunResponse{RunID: 999}
	})

	w := &databricks.WorkspaceClient{Config: &config.Config{Host: server.URL, Token: "token"}}
	jc, err := newJobsSubmitClient(w)
	require.NoError(t, err)

	runID, err := jc.submit(t.Context(), jobsSubmitRun{RunName: "exp", Tasks: []submitTask{{TaskKey: "exp"}}})
	require.NoError(t, err)
	assert.Equal(t, int64(999), runID)
	assert.Equal(t, "exp", got.RunName)
}

func TestSubmitWorkload(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	testserver.AddDefaultHandlers(server)

	var got jobsSubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return submitRunResponse{RunID: 777}
	})
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
	require.Len(t, at.Deployments, 1)
	d := at.Deployments[0]
	assert.True(t, strings.HasSuffix(d.CommandPath, "/"+commandScriptName), d.CommandPath)
	assert.Contains(t, d.CommandPath, "/.air/cli_launch/")
	assert.Equal(t, aiRuntimeCompute{AcceleratorType: "GPU_1xH100", AcceleratorCount: 1}, d.Compute)
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

	t.Run("code_source rejected", func(t *testing.T) {
		cfg := *base
		cfg.CodeSource = &codeSourceConfig{Type: "snapshot"}
		_, _, err := submitWorkload(t.Context(), w, &cfg, cfgPath, "")
		require.ErrorContains(t, err, "code_source is not yet supported")
	})
}
