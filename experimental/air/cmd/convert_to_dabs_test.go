package aircmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToDabsCommandShape(t *testing.T) {
	cmd := newConvertToDabsCommand()
	assert.Equal(t, "convert-to-dabs <yaml_path>", cmd.Use)
	assert.Empty(t, cmd.Commands(), "convert-to-dabs must not register subcommands")
	// Exactly one positional (the YAML path).
	assert.NoError(t, cmd.Args(cmd, []string{"run.yaml"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"a", "b"}))
}

// get is a small helper: read a dotted path out of the emitted bundle root.
func get(t *testing.T, root map[string]dyn.Value, path string) dyn.Value {
	t.Helper()
	v, err := dyn.GetByPath(dyn.V(root), dyn.MustPathFromString(path))
	require.NoError(t, err, "path %q should exist", path)
	return v
}

func has(root map[string]dyn.Value, path string) bool {
	_, err := dyn.GetByPath(dyn.V(root), dyn.MustPathFromString(path))
	return err == nil
}

// A full config maps onto a schema-shaped bundle: bundle name, job/task keys, the
// ai_runtime_task (experiment + single deployment + code_source_path), framework
// fields on the task wrapper, and the environment spec.
func TestConvertToDabsFullMapping(t *testing.T) {
	cfg := minimalConfig + `
max_retries: 2
timeout_minutes: 30
mlflow_run_name: run-42
code_source:
  type: snapshot
  snapshot:
    root_path: ./src
environment:
  version: 5
  dependencies:
    - numpy
    - torch
`
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, artifacts, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)

	name := loaded.ExperimentName
	assert.Equal(t, name, get(t, root, "bundle.name").MustString())
	assert.Equal(t, "development", get(t, root, "targets.dev.mode").MustString())

	jobPath := "resources.jobs." + name
	assert.Equal(t, name, get(t, root, jobPath+".name").MustString())

	task := jobPath + ".tasks[0]"
	assert.Equal(t, name, get(t, root, task+".task_key").MustString())
	// Framework fields live on the task wrapper, not in ai_runtime_task.
	assert.Equal(t, int64(2), get(t, root, task+".max_retries").MustInt())
	assert.Equal(t, int64(1800), get(t, root, task+".timeout_seconds").MustInt())
	assert.False(t, has(root, task+".ai_runtime_task.max_retries"), "retries must not be inside ai_runtime_task")

	art := task + ".ai_runtime_task"
	assert.Equal(t, name, get(t, root, art+".experiment").MustString())
	assert.Equal(t, "run-42", get(t, root, art+".mlflow_run").MustString())
	// code_source_path is a local directory (packaged by the deploy-time aicode
	// mutator), not a pre-built tarball.
	assert.Equal(t, "./"+codeSourceDirName, get(t, root, art+".code_source_path").MustString())

	dep := art + ".deployments[0]"
	assert.Equal(t, "./"+commandScriptName, get(t, root, dep+".command_path").MustString())
	assert.Equal(t, "GPU_1xH100", get(t, root, dep+".compute.accelerator_type").MustString())
	assert.Equal(t, int64(1), get(t, root, dep+".compute.accelerator_count").MustInt())

	env := jobPath + ".environments[0]"
	assert.Equal(t, "default", get(t, root, env+".environment_key").MustString())
	assert.Equal(t, "5", get(t, root, env+".spec.environment_version").MustString())
	deps := get(t, root, env+".spec.dependencies").MustSequence()
	require.Len(t, deps, 2)
	assert.Equal(t, "numpy", deps[0].MustString())

	// command.sh is always an artifact. requirements.yaml is NOT emitted: the
	// deploy-time aicode.SynthesizeRequirements mutator regenerates it from the
	// environments[] spec (asserted above), so convert must not also write it.
	assert.Contains(t, itemNames(artifacts), commandScriptName)
	assert.NotContains(t, itemNames(artifacts), requirementsName)
}

// Optional fields are omitted rather than emitted empty: no code_source means no
// code_source_path; unset retries/timeout means no wrapper fields.
func TestConvertToDabsOmitsUnsetFields(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", minimalConfig)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)

	name := loaded.ExperimentName
	task := "resources.jobs." + name + ".tasks[0]"
	assert.False(t, has(root, task+".max_retries"))
	assert.False(t, has(root, task+".timeout_seconds"))
	assert.False(t, has(root, task+".ai_runtime_task.code_source_path"))
	assert.False(t, has(root, task+".ai_runtime_task.mlflow_run"))
	// The command still needs a home even without code_source.
	assert.Equal(t, "./"+commandScriptName, get(t, root, task+".ai_runtime_task.deployments[0].command_path").MustString())

	// Even with no environment block, the default runtime version is pinned (what
	// `air run` would have used) rather than emitting an empty environment spec.
	env := "resources.jobs." + name + ".environments[0]"
	assert.Equal(t, "4", get(t, root, env+".spec.environment_version").MustString())
	assert.False(t, has(root, env+".spec.dependencies"))
}

// A DATABRICKS_DL_RUNTIME_IMAGE env override flows through the same resolution
// `air run` uses, so a converted bundle pins the same version.
func TestConvertToDabsRuntimeVersionEnvOverride(t *testing.T) {
	t.Setenv(dlRuntimeImageEnv, "CLIENT-GPU-7")
	path := writeConfigFile(t, "run.yaml", minimalConfig)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)

	env := "resources.jobs." + loaded.ExperimentName + ".environments[0]"
	assert.Equal(t, "7", get(t, root, env+".spec.environment_version").MustString())
}

// remote_volume can't be honored by a converted bundle (bundle deploy owns the
// artifact upload location), so it is rejected rather than silently ignored.
func TestConvertToDabsRejectsRemoteVolume(t *testing.T) {
	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ./src
    remote_volume: /Volumes/main/default/code
`
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)
	_, _, err = convertToDabs(t.Context(), loaded, path)
	require.ErrorContains(t, err, "remote_volume is not supported")
}

// env_variables / secrets / parameters ride as sidecar files (the ai_runtime_task
// proto has no inline fields for them), matching the CLI's own launch layout.
func TestConvertToDabsStagesEnvAndSecretSidecars(t *testing.T) {
	cfg := minimalConfig + `
env_variables:
  FOO: bar
secrets:
  TOKEN: scope/key
parameters:
  lr: 0.1
`
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, artifacts, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)

	names := itemNames(artifacts)
	assert.Contains(t, names, envVarsName)
	assert.Contains(t, names, secretEnvVarsName)
	assert.Contains(t, names, hyperparametersName)

	// They are NOT smuggled into ai_runtime_task (which would fail bundle validate).
	name := loaded.ExperimentName
	art := "resources.jobs." + name + ".tasks[0].ai_runtime_task"
	assert.False(t, has(root, art+".env_variables"))
	assert.False(t, has(root, art+".secrets"))
}

// A working-tree (non-git-pinned) code_source is staged into the bundle's
// code_source/ directory, honoring .gitignore just like the run path's plain-tar.
func TestConvertToDabsWorkingTreeMaterialized(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "train.py", "print('x')")
	writeRepoFile(t, repo, "notes.log", "scratch")
	writeRepoFile(t, repo, ".gitignore", "*.log\n")

	cfg := "experiment_name: wt\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: " + repo + "\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	outDir := t.TempDir()
	_, err = writeBundle(t.Context(), loaded, path, outDir)
	require.NoError(t, err)

	codeDir := filepath.Join(outDir, codeSourceDirName)
	assert.FileExists(t, filepath.Join(codeDir, "train.py"))
	assert.NoFileExists(t, filepath.Join(codeDir, "notes.log"), "gitignored files must be excluded from the staged code_source")
}

// A requirements-FILE dependency set (environment.dependencies is a path) is folded
// into the environments[] spec so the deploy-time aicode mutator can regenerate
// requirements.yaml from it. Convert emits no requirements.yaml artifact of its own.
func TestConvertToDabsFoldsRequirementsFileIntoEnvSpec(t *testing.T) {
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "requirements.yaml")
	require.NoError(t, os.WriteFile(reqPath, []byte("version: \"6\"\ndependencies:\n  - numpy\n  - pandas\n"), 0o600))

	cfg := "experiment_name: reqfile\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"environment:\n  dependencies: " + reqPath + "\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, artifacts, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)

	env := "resources.jobs." + loaded.ExperimentName + ".environments[0]"
	assert.Equal(t, "6", get(t, root, env+".spec.environment_version").MustString())
	deps := get(t, root, env+".spec.dependencies").MustSequence()
	require.Len(t, deps, 2)
	assert.Equal(t, "numpy", deps[0].MustString())
	assert.Equal(t, "pandas", deps[1].MustString())

	// No requirements.yaml artifact: the mutator regenerates it from the spec.
	assert.NotContains(t, itemNames(artifacts), requirementsName)
}

// conversionNotes surfaces what was transformed/staged so a migrating user knows
// what changed between their run YAML and the bundle.
func TestConvertToDabsConversionNotes(t *testing.T) {
	cfg := minimalConfig + `
env_variables: {FOO: bar}
secrets: {TOKEN: scope/key}
parameters: {lr: 0.1}
code_source:
  type: snapshot
  snapshot:
    root_path: ./src
    git: {commit: abc123}
`
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	notes := conversionNotes(loaded)
	joined := strings.Join(notes, "\n")
	assert.Contains(t, joined, "code_source.git")      // git pin flagged
	assert.Contains(t, joined, "code_source/")         // staged-directory behavior
	assert.Contains(t, joined, "env_vars.json")        // env vars staged
	assert.Contains(t, joined, "secret_env_vars.json") // secrets staged
	assert.Contains(t, joined, "hyperparameters.yaml") // parameters staged

	// A minimal config with none of those has no notes.
	base := writeConfigFile(t, "min.yaml", minimalConfig)
	minCfg, err := loadRunConfig(base)
	require.NoError(t, err)
	assert.Empty(t, conversionNotes(minCfg))
}

// usage_policy_id is a resolved budget policy id and maps to the job's
// budget_policy_id (usage_policy_name, which needs resolution, is rejected).
func TestConvertToDabsMapsUsagePolicyID(t *testing.T) {
	cfg := minimalConfig + "usage_policy_id: budget-abc-123\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)
	assert.Equal(t, "budget-abc-123", get(t, root, "resources.jobs."+loaded.ExperimentName+".budget_policy_id").MustString())
}

func TestConvertToDabsMapsPermissions(t *testing.T) {
	cfg := minimalConfig + `
permissions:
  - user_name: alice@example.com
    level: CAN_MANAGE
  - group_name: eng
    level: CAN_VIEW
`
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)

	perms := get(t, root, "resources.jobs."+loaded.ExperimentName+".permissions").MustSequence()
	require.Len(t, perms, 2)
	assert.Equal(t, "CAN_MANAGE", perms[0].Get("level").MustString())
	assert.Equal(t, "alice@example.com", perms[0].Get("user_name").MustString())
	assert.Equal(t, "eng", perms[1].Get("group_name").MustString())
}

// A git-pinned code_source is materialized from the commit, not the dirty working
// tree: writeBundle stages a code_source/ directory holding the committed file only.
func TestConvertToDabsGitPinnedMaterialized(t *testing.T) {
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print('committed')")
	sha := commitAll(t, repo, "init")
	// Dirty the tree AFTER the commit; the pinned snapshot must not include this.
	writeRepoFile(t, repo, "uncommitted.py", "print('dirty')")

	cfg := "experiment_name: git-pin\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: " + repo + "\n    git:\n      commit: " + sha + "\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	outDir := t.TempDir()
	_, err = writeBundle(t.Context(), loaded, path, outDir)
	require.NoError(t, err)

	codeDir := filepath.Join(outDir, codeSourceDirName)
	assert.FileExists(t, filepath.Join(codeDir, "train.py"))
	assert.NoFileExists(t, filepath.Join(codeDir, "uncommitted.py"), "git-pinned snapshot must exclude uncommitted files")
}

// git archive runs with `git -C repoPath`, so the staging tarball path must be
// absolute — otherwise `-o out/...` resolves against the repo dir and fails. Exercise
// writeBundle from a working dir with a RELATIVE output path against a git-pinned source.
func TestConvertToDabsGitPinnedRelativeOutputDir(t *testing.T) {
	repo := newTestRepo(t)
	writeRepoFile(t, repo, "train.py", "print('x')")
	sha := commitAll(t, repo, "init")

	cfg := "experiment_name: rel-out\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: " + repo + "\n    git:\n      commit: " + sha + "\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	// chdir into a scratch dir and pass a relative --output-dir.
	work := t.TempDir()
	t.Chdir(work)
	_, err = writeBundle(t.Context(), loaded, path, "out")
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(work, "out", codeSourceDirName, "train.py"))
}

// A numeric or reserved-word experiment_name would be an unquoted YAML map key
// that DABs' strict loader rejects (!!int / !!bool). The job resource key is
// prefixed to stay a string, while name/experiment keep the original value.
func TestConvertToDabsSafeJobKey(t *testing.T) {
	cases := map[string]string{
		"12345": "job_12345",
		"1.5e3": "job_1.5e3",
		"true":  "job_true",
		"null":  "job_null",
	}
	for name, wantKey := range cases {
		assert.Equal(t, wantKey, bundleResourceKey(name), "key for %q", name)
	}
	// A normal name is used as-is.
	assert.Equal(t, "my-run_1", bundleResourceKey("my-run_1"))

	// End to end: a numeric name lands under the prefixed key, but name/experiment
	// keep the numeric string value.
	cfg := "experiment_name: \"12345\"\ncommand: python t.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)
	root, _, err := convertToDabs(t.Context(), loaded, path)
	require.NoError(t, err)
	assert.Equal(t, "12345", get(t, root, "resources.jobs.job_12345.name").MustString())
	assert.Equal(t, "12345", get(t, root, "resources.jobs.job_12345.tasks[0].ai_runtime_task.experiment").MustString())
}

func TestConvertToDabsRejectsUnsupported(t *testing.T) {
	t.Run("docker_image", func(t *testing.T) {
		cfg := minimalConfig + `
environment:
  docker_image:
    url: myregistry/img:tag
`
		path := writeConfigFile(t, "run.yaml", cfg)
		loaded, err := loadRunConfig(path)
		require.NoError(t, err)
		_, _, err = convertToDabs(t.Context(), loaded, path)
		require.ErrorContains(t, err, "docker_image is not yet supported")
	})

	t.Run("usage_policy_name", func(t *testing.T) {
		cfg := minimalConfig + "usage_policy_name: my-policy\n"
		path := writeConfigFile(t, "run.yaml", cfg)
		loaded, err := loadRunConfig(path)
		require.NoError(t, err)
		_, _, err = convertToDabs(t.Context(), loaded, path)
		require.ErrorContains(t, err, "usage_policy_name is not yet supported")
	})
}
