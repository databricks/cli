package aircmd

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubValidateConfig registers an OK ValidateConfig response so submitWorkload's
// pre-flight passes. Register before AddDefaultHandlers (the router is first-wins).
func stubValidateConfig(server *testserver.Server) {
	server.Handle("POST", "/api/2.0/ai-training/config:validate", func(req testserver.Request) any {
		return validateConfigResponse{}
	})
}

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

	p := buildSubmitPayload(cfg, "/d/command.sh", "5", "", snapshotResult{}, nil)

	assert.Equal(t, "exp", p.RunName)
	assert.Equal(t, 1800, p.TimeoutSeconds)
	// No policy configured: the field stays empty and is omitted from the wire form.
	assert.Empty(t, p.BudgetPolicyId)
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
	task := buildSubmitPayload(cfg, "/d/command.sh", "4", "", snapshotResult{}, nil).Tasks[0]
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
	task := buildSubmitPayload(cfg, "/d/command.sh", "4", "", snapshotResult{}, nil).Tasks[0]
	assert.Equal(t, 0, task.MaxRetries)
	assert.False(t, task.RetryOnTimeout)

	b, err := json.Marshal(task)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"max_retries":0`)
	assert.NotContains(t, string(b), "retry_on_timeout")
}

// TestBuildSubmitPayloadInlineDependencies covers how deps land on the environment
// spec: a non-empty list is set alongside the runtime channel; empty and nil omit
// the key so the payload is unchanged.
func TestBuildSubmitPayloadInlineDependencies(t *testing.T) {
	cfg := &runConfig{
		ExperimentName: "exp",
		Command:        new("x"),
		Compute:        &computeConfig{AcceleratorType: "GPU_8xH100", NumAccelerators: 8},
	}

	deps := []string{"torch==2.3.0", "--extra-index-url https://internal/pypi", "numpy"}
	spec := buildSubmitPayload(cfg, "/d/command.sh", "5", "", snapshotResult{}, deps).Environments[0].Spec
	assert.Equal(t, deps, spec.Dependencies)
	assert.Equal(t, "5", spec.EnvironmentVersion)

	// The SDK marshaler drops empty/nil slices, so no "dependencies" key is emitted.
	for _, empty := range [][]string{{}, nil} {
		spec = buildSubmitPayload(cfg, "/d/command.sh", "5", "", snapshotResult{}, empty).Environments[0].Spec
		b, err := json.Marshal(spec)
		require.NoError(t, err)
		assert.NotContains(t, string(b), "dependencies")
	}
}

// TestEnvironmentDependencies covers how declared deps are resolved to a flat list:
// an inline list (no file version), a requirements file (path resolved against the
// config dir, version read from the file), none, and a missing file.
func TestEnvironmentDependencies(t *testing.T) {
	inline := &runConfig{Environment: &environmentConfig{
		Dependencies: dependencies{set: true, isList: true, list: []string{"torch", "numpy"}},
	}}
	deps, version, err := environmentDependencies(inline, "run.yaml")
	require.NoError(t, err)
	assert.Equal(t, []string{"torch", "numpy"}, deps)
	assert.Empty(t, version)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reqs.yaml"), []byte("version: \"5\"\ndependencies:\n  - pandas\n"), 0o600))
	fromFile := &runConfig{Environment: &environmentConfig{
		Dependencies: dependencies{set: true, isList: false, path: "reqs.yaml"},
	}}
	deps, version, err = environmentDependencies(fromFile, filepath.Join(dir, "run.yaml"))
	require.NoError(t, err)
	assert.Equal(t, []string{"pandas"}, deps)
	assert.Equal(t, "5", version)

	deps, _, err = environmentDependencies(&runConfig{}, "run.yaml")
	require.NoError(t, err)
	assert.Nil(t, deps)

	missing := &runConfig{Environment: &environmentConfig{
		Dependencies: dependencies{set: true, isList: false, path: "nope.yaml"},
	}}
	_, _, err = environmentDependencies(missing, filepath.Join(dir, "run.yaml"))
	require.ErrorContains(t, err, "failed to read requirements file")
}

// TestReadRequirementsDependencies covers reading a requirements file's dependency
// list and version, with a missing key yielding an empty list and a -r include
// rejected.
func TestReadRequirementsDependencies(t *testing.T) {
	dir := t.TempDir()

	reqPath := filepath.Join(dir, "requirements.yaml")
	require.NoError(t, os.WriteFile(reqPath, []byte("version: \"5\"\ndependencies:\n  - torch==2.3.0\n  - numpy\n"), 0o600))
	deps, version, err := readRequirementsDependencies(reqPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"torch==2.3.0", "numpy"}, deps)
	assert.Equal(t, "5", version)

	emptyPath := filepath.Join(dir, "empty.yaml")
	require.NoError(t, os.WriteFile(emptyPath, []byte("version: \"5\"\n"), 0o600))
	deps, _, err = readRequirementsDependencies(emptyPath)
	require.NoError(t, err)
	assert.Empty(t, deps)

	includePath := filepath.Join(dir, "include.yaml")
	require.NoError(t, os.WriteFile(includePath, []byte("dependencies:\n  - -r other.txt\n"), 0o600))
	_, _, err = readRequirementsDependencies(includePath)
	require.ErrorContains(t, err, "requirements-file include")
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
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cfg, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	runID, dashboardURL, err := submitWorkload(t.Context(), w, cfg, cfgPath, "idem-key", false)
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

// TestSubmitWorkloadHonorsOverride proves a --override reaches the actual
// runs/submit payload on a real submit, not just dry-run validation: the config
// pins num_accelerators=1, the override bumps it to 4, and the recorded request
// body must carry 4.
func TestSubmitWorkloadHonorsOverride(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	// Register before AddDefaultHandlers: the router is first-wins, so this must
	// claim the route ahead of the default jobs/runs/submit handler.
	var got jobs.SubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return jobs.SubmitRunResponse{RunId: 777}
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	cfg, err := loadRunConfigWithOverrides(t.Context(), cfgPath, []string{"compute.num_accelerators=4"})
	require.NoError(t, err)

	_, _, err = submitWorkload(t.Context(), w, cfg, cfgPath, "idem-key", false)
	require.NoError(t, err)

	require.Len(t, got.Tasks, 1)
	at := got.Tasks[0].AiRuntimeTask
	require.NotNil(t, at)
	require.Len(t, at.Deployments, 1)
	assert.Equal(t, 4, at.Deployments[0].Compute.AcceleratorCount)
}

// A working-tree code_source is packaged into a tarball, uploaded via DABs' artifact
// plumbing, and its remote code_source_path attached to the submitted task.
func TestSubmitWorkloadWithCodeSource(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	// Register before AddDefaultHandlers: the router is first-wins, so this must claim the route ahead of the default handler.
	var got jobs.SubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return jobs.SubmitRunResponse{RunId: 555}
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	// A plain working-tree directory: packaging is plain-tar.
	repo := filepath.Join(t.TempDir(), "src")
	writeRepoFile(t, repo, "train.py", "print()")

	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ` + repo + `
`
	cfgPath := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	// The DABs upload path logs via cmdio; the real `air run` context carries it.
	ctx := cmdio.MockDiscard(t.Context())
	_, _, err = submitWorkload(ctx, w, loaded, cfgPath, "idem", false)
	require.NoError(t, err)

	at := got.Tasks[0].AiRuntimeTask
	// The tarball is uploaded to the artifact .internal dir and code_source_path
	// rewritten to it.
	assert.Contains(t, at.CodeSourcePath, "/.air/repo_snapshots/.internal/")
	assert.True(t, strings.HasSuffix(at.CodeSourcePath, ".tar.gz"), at.CodeSourcePath)
}

// A git-pinned code_source is git-archived at the commit, uploaded via DABs' artifact
// plumbing, and its remote code_source_path attached to the submitted task.
func TestSubmitWorkloadWithGitPinnedCodeSource(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var got jobs.SubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return jobs.SubmitRunResponse{RunId: 555}
	})
	stubValidateConfig(server)
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

	ctx := cmdio.MockDiscard(t.Context())
	_, _, err = submitWorkload(ctx, w, loaded, cfgPath, "idem", false)
	require.NoError(t, err)

	at := got.Tasks[0].AiRuntimeTask
	assert.Contains(t, at.CodeSourcePath, "/.air/repo_snapshots/.internal/")
	assert.True(t, strings.HasSuffix(at.CodeSourcePath, ".tar.gz"), at.CodeSourcePath)
}

// testSidecarStore builds a workspace filer + base path standing in for the run's
// launch dir, where snapshotViaDABsUpload writes git provenance sidecars.
func testSidecarStore(t *testing.T, w *databricks.WorkspaceClient) (filer.Filer, string) {
	t.Helper()
	base := "/Workspace/Users/tester@databricks.com/.air/cli_launch/test"
	f, err := filer.NewWorkspaceFilesClient(w, base)
	require.NoError(t, err)
	return f, base
}

// A plain-tar (working-tree) snapshot is uploaded under a unique, timestamped name so
// two concurrent submissions of the same root_path don't clobber each other's upload.
func TestSubmitWorkloadPlainTarNameIsUnique(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		return jobs.SubmitRunResponse{RunId: 555}
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	// A plain working-tree directory named "src": the old code named the tarball
	// after the dir alone (src.tar.gz), so any two submissions collided.
	repo := filepath.Join(t.TempDir(), "src")
	writeRepoFile(t, repo, "train.py", "print()")

	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ` + repo + `
`
	cfgPath := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	// The uploaded name carries a discriminator (timestamp), not the bare dir name.
	ctx := cmdio.MockDiscard(t.Context())
	sidecarStore, sidecarBase := testSidecarStore(t, w)
	snap, err := snapshotViaDABsUpload(ctx, w, loaded.CodeSource.Snapshot, cfgPath, sidecarStore, sidecarBase)
	require.NoError(t, err)
	base := path.Base(snap.CodeSourcePath)
	assert.NotEqual(t, "src.tar.gz", base, "plain-tar name must be unique, not the bare dir name")
	assert.Regexp(t, `^src_\d{8}_\d{6}\.tar\.gz$`, base)
}

// A git_archive snapshot is content-addressed by (commit, include_paths): submitting
// the same commit twice reuses the already-uploaded tarball and skips the second
// upload (cache hit), while resolving to the identical remote path.
func TestSubmitWorkloadGitArchiveCaching(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		return jobs.SubmitRunResponse{RunId: 555}
	})
	// Track which snapshot tarballs get uploaded, preserving fake-workspace
	// persistence so the second submit's cache-existence Stat sees the first upload.
	// Dedupe by path: the DABs uploader mkdirs-and-retries the import on a missing
	// parent dir, so one logical upload can hit this route more than once.
	uploaded := map[string]bool{}
	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req testserver.Request) any {
		p := req.Vars["path"]
		if strings.Contains(p, "/.air/repo_snapshots/") {
			uploaded[p] = true
		}
		return req.Workspace.WorkspaceFilesImportFile(p, req.Body, req.URL.Query().Get("overwrite") == "true")
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

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

	ctx := cmdio.MockDiscard(t.Context())
	sidecarStore, sidecarBase := testSidecarStore(t, w)
	first, err := snapshotViaDABsUpload(ctx, w, loaded.CodeSource.Snapshot, cfgPath, sidecarStore, sidecarBase)
	require.NoError(t, err)
	second, err := snapshotViaDABsUpload(ctx, w, loaded.CodeSource.Snapshot, cfgPath, sidecarStore, sidecarBase)
	require.NoError(t, err)

	// Same pinned commit → identical content-addressed remote path, uploaded once
	// (the second submit is a cache hit and moves no bytes).
	assert.Equal(t, first.CodeSourcePath, second.CodeSourcePath)
	assert.Len(t, uploaded, 1, "git_archive cache hit should skip the second upload")
}

// A git code_source also uploads git provenance sidecars (git_state.json, and
// git_diff.patch when the tree is dirty) next to the run's launch dir, so the
// submitted commit + working-tree diff are inspectable.
func TestSubmitWorkloadUploadsGitSidecars(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		return jobs.SubmitRunResponse{RunId: 555}
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	// Commit, then dirty the tree so both git_state.json and git_diff.patch are produced.
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print()")
	commitAll(t, repo, "init")
	writeRepoFile(t, repo, "train.py", "print('dirty')")

	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ` + repo + `
`
	cfgPath := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	sidecarStore, sidecarBase := testSidecarStore(t, w)
	snap, err := snapshotViaDABsUpload(ctx, w, loaded.CodeSource.Snapshot, cfgPath, sidecarStore, sidecarBase)
	require.NoError(t, err)

	// Both sidecars are reported under the launch dir and actually exist there.
	assert.Equal(t, path.Join(sidecarBase, gitStateName), snap.GitStatePath)
	assert.Equal(t, path.Join(sidecarBase, gitDiffName), snap.GitDiffPath)

	r, err := sidecarStore.Read(ctx, gitStateName)
	require.NoError(t, err)
	stateBytes, err := io.ReadAll(r)
	require.NoError(t, err)
	var state map[string]any
	require.NoError(t, json.Unmarshal(stateBytes, &state))
	assert.Equal(t, "plain_tar", state["packaging_mode"])
	assert.Equal(t, true, state["dirty"])
	assert.Equal(t, "captured", state["diff_status"])
}

// remote_volume uploads the snapshot to a UC Volume: DABs' artifact uploader handles
// /Volumes destinations natively, so code_source_path lands under the Volume path.
func TestSubmitWorkloadWithRemoteVolumeCodeSource(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var got jobs.SubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return jobs.SubmitRunResponse{RunId: 555}
	})
	// Stub the UC Volume file write: the fake server's default handler 404s when the
	// parent dir is absent (no auto-mkdir), so accept the PUT to exercise the Volume
	// upload route. This asserts we route to /api/2.0/fs/files/Volumes/... at all.
	server.Handle("PUT", "/api/2.0/fs/files/Volumes/{path...}", func(req testserver.Request) any {
		return testserver.Response{StatusCode: 204}
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	repo := filepath.Join(t.TempDir(), "src")
	writeRepoFile(t, repo, "train.py", "print()")

	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ` + repo + `
    remote_volume: /Volumes/main/default/code
`
	cfgPath := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	_, _, err = submitWorkload(ctx, w, loaded, cfgPath, "idem", false)
	require.NoError(t, err)

	at := got.Tasks[0].AiRuntimeTask
	assert.Contains(t, at.CodeSourcePath, "/Volumes/main/default/code/.internal/")
	assert.True(t, strings.HasSuffix(at.CodeSourcePath, ".tar.gz"), at.CodeSourcePath)
}

func TestSubmitWorkloadGuards(t *testing.T) {
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig)
	base, err := loadRunConfig(cfgPath)
	require.NoError(t, err)

	t.Run("unresolvable usage_policy_name fails before upload", func(t *testing.T) {
		// An empty policy list makes the name unresolvable. Record every path the
		// server sees so the "fails before any upload" ordering is asserted, not just
		// asserted-by-comment: no import/mkdirs request may be made.
		server := testserver.New(t)
		t.Cleanup(server.Close)
		var paths []string
		server.Handle("GET", "/api/2.0/serverless-policies", func(req testserver.Request) any {
			paths = append(paths, req.URL.Path)
			return usagePoliciesResponse{}
		})
		server.Handle("POST", "/api/2.0/workspace/{path...}", func(req testserver.Request) any {
			paths = append(paths, req.URL.Path)
			return testserver.Response{StatusCode: 200}
		})
		stubValidateConfig(server)
		testserver.AddDefaultHandlers(server)
		pw, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
		require.NoError(t, err)

		cfg := *base
		cfg.UsagePolicyName = new("nope")
		_, _, err = submitWorkload(t.Context(), pw, &cfg, cfgPath, "", false)
		require.ErrorContains(t, err, `no usage policy named "nope"`)
		for _, p := range paths {
			assert.NotContains(t, p, "/workspace/", "no workspace write may precede policy resolution")
		}
	})

	t.Run("bad requirements file fails before any upload", func(t *testing.T) {
		server := testserver.New(t)
		t.Cleanup(server.Close)
		var uploaded bool
		server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(testserver.Request) any {
			uploaded = true
			return nil
		})
		stubValidateConfig(server)
		testserver.AddDefaultHandlers(server)
		tw, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
		require.NoError(t, err)

		cfg := *base
		cfg.Environment = &environmentConfig{Dependencies: dependencies{set: true, isList: false, path: "missing.yaml"}}
		_, _, err = submitWorkload(t.Context(), tw, &cfg, cfgPath, "", false)
		require.ErrorContains(t, err, "failed to read requirements file")
		assert.False(t, uploaded, "no artifacts should be uploaded when dependency resolution fails")
	})
}

// The resolved policy id must reach the submit payload, by literal id and by name.
func TestSubmitWorkloadSendsUsagePolicy(t *testing.T) {
	const policyID = "12345678-90ab-cdef-1234-567890abcdef"

	setup := func(t *testing.T) (*databricks.WorkspaceClient, *jobs.SubmitRun) {
		server := testserver.New(t)
		t.Cleanup(server.Close)

		got := &jobs.SubmitRun{}
		server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
			require.NoError(t, json.Unmarshal(req.Body, got))
			return jobs.SubmitRunResponse{RunId: 1}
		})
		server.Handle("GET", "/api/2.0/serverless-policies", func(req testserver.Request) any {
			return usagePoliciesResponse{Policies: []usagePolicy{{PolicyID: policyID, PolicyName: "team-a"}}}
		})
		stubValidateConfig(server)
		testserver.AddDefaultHandlers(server)
		w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
		require.NoError(t, err)
		return w, got
	}

	t.Run("literal id", func(t *testing.T) {
		w, got := setup(t)
		cfgPath := writeConfigFile(t, "run.yaml", minimalConfig+"usage_policy_id: "+policyID+"\n")
		cfg, err := loadRunConfig(cfgPath)
		require.NoError(t, err)

		_, _, err = submitWorkload(cmdio.MockDiscard(t.Context()), w, cfg, cfgPath, "idem", false)
		require.NoError(t, err)
		assert.Equal(t, policyID, got.BudgetPolicyId)
	})

	t.Run("resolved from name", func(t *testing.T) {
		w, got := setup(t)
		cfgPath := writeConfigFile(t, "run.yaml", minimalConfig+"usage_policy_name: team-a\n")
		cfg, err := loadRunConfig(cfgPath)
		require.NoError(t, err)

		_, _, err = submitWorkload(cmdio.MockDiscard(t.Context()), w, cfg, cfgPath, "idem", false)
		require.NoError(t, err)
		assert.Equal(t, policyID, got.BudgetPolicyId)
	})
}
