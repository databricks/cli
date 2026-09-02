package aircmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// Re-running a conversion into a dir that already holds a generated bundle is
// refused by default (the user may have edited it) and allowed with --force.
func TestConvertToDabsForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o700))
	cfg := "experiment_name: overwrite\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: ./src\n"
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	_, err = writeBundle(t.Context(), loaded, path, dir, false)
	require.NoError(t, err)

	// A hand edit must survive a refused re-run.
	edited := []byte("# hand-edited\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), edited, 0o600))

	_, err = writeBundle(t.Context(), loaded, path, dir, false)
	require.ErrorContains(t, err, "pass --force to overwrite")
	kept, err := os.ReadFile(filepath.Join(dir, "databricks.yml"))
	require.NoError(t, err)
	assert.Equal(t, edited, kept)

	// With --force the generated bundle replaces it.
	_, err = writeBundle(t.Context(), loaded, path, dir, true)
	require.NoError(t, err)
	regenerated, err := os.ReadFile(filepath.Join(dir, "databricks.yml"))
	require.NoError(t, err)
	assert.NotEqual(t, edited, regenerated)
	assert.Contains(t, string(regenerated), "ai_runtime_task")
}

// command.sh is the user's command copied verbatim — convert injects no cd. The
// launcher exports $CODE_SOURCE_PATH and runs the command in the launch dir without
// cd-ing, so (as under air run) the command owns its own cd; convert must not add one,
// which would make it more lenient than air run.
func TestConvertToDabsCommandVerbatim(t *testing.T) {
	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ./src
`
	path := writeConfigFile(t, "run.yaml", cfg)
	require.NoError(t, os.MkdirAll(filepath.Join(filepath.Dir(path), "src"), 0o700))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	_, artifacts, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)

	var cmd string
	for _, it := range artifacts {
		if it.name == commandScriptName {
			cmd = string(it.data)
		}
	}
	assert.Equal(t, "python train.py", cmd, "command.sh must be the verbatim command, no injected cd")
	assert.NotContains(t, cmd, "cd /databricks/code_source")
}

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
	require.NoError(t, os.MkdirAll(filepath.Join(filepath.Dir(path), "src"), 0o700))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, artifacts, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
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
	// code_source_path is the source dir relative to the bundle; the deploy-time
	// aicode mutator packages it in place.
	assert.Equal(t, "./src", get(t, root, art+".code_source_path").MustString())

	dep := art + ".deployments[0]"
	assert.Equal(t, "./"+generatedArtifactsDir+"/"+commandScriptName, get(t, root, dep+".command_path").MustString())
	assert.Equal(t, "GPU_1xH100", get(t, root, dep+".compute.accelerator_type").MustString())
	assert.Equal(t, int64(1), get(t, root, dep+".compute.accelerator_count").MustInt())

	env := jobPath + ".environments[0]"
	assert.Equal(t, "default", get(t, root, env+".environment_key").MustString())
	assert.Equal(t, "5", get(t, root, env+".spec.environment_version").MustString())
	deps := get(t, root, env+".spec.dependencies").MustSequence()
	require.Len(t, deps, 2)
	assert.Equal(t, "numpy", deps[0].MustString())

	// command.sh and training_config.yaml are always artifacts; the Jobs run-output
	// page derives the latter's path from command_path. requirements.yaml is not
	// emitted — deps ride on the environments[] spec asserted above.
	assert.Contains(t, itemNames(artifacts), commandScriptName)
	assert.Contains(t, itemNames(artifacts), trainingConfigName)
	assert.NotContains(t, itemNames(artifacts), "requirements.yaml")
}

// Optional fields are omitted rather than emitted empty: no code_source means no
// code_source_path; unset retries/timeout means no wrapper fields.
// sync.paths lists only the generated-artifacts dir. The code directory must be
// absent: deploy packages it into the snapshot tarball, so syncing it as loose
// files too would upload the whole tree a second time.
func TestConvertToDabsSyncPathsExcludesCodeDir(t *testing.T) {
	cfg := minimalConfig + `
code_source:
  type: snapshot
  snapshot:
    root_path: ./src
`
	path := writeConfigFile(t, "run.yaml", cfg)
	require.NoError(t, os.MkdirAll(filepath.Join(filepath.Dir(path), "src"), 0o700))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)

	paths := get(t, root, "sync.paths").MustSequence()
	require.Len(t, paths, 1)
	assert.Equal(t, generatedArtifactsDir, paths[0].MustString())

	// The task still points at the code dir; only the sync set omits it.
	task := "resources.jobs." + loaded.ExperimentName + ".tasks[0].ai_runtime_task"
	assert.Equal(t, "./src", get(t, root, task+".code_source_path").MustString())
}

func TestConvertToDabsOmitsUnsetFields(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", minimalConfig)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)

	name := loaded.ExperimentName
	task := "resources.jobs." + name + ".tasks[0]"
	// max_retries is always emitted: `air run` fills its own default when unset, so
	// omitting it would silently give the bundle the Jobs default instead.
	assert.Equal(t, int64(defaultMaxRetries), get(t, root, task+".max_retries").MustInt())
	assert.False(t, has(root, task+".timeout_seconds"))
	assert.False(t, has(root, task+".ai_runtime_task.code_source_path"))
	assert.False(t, has(root, task+".ai_runtime_task.mlflow_run"))
	// The command still needs a home even without code_source.
	assert.Equal(t, "./"+generatedArtifactsDir+"/"+commandScriptName, get(t, root, task+".ai_runtime_task.deployments[0].command_path").MustString())

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

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
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
	_, _, err = convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
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

	root, artifacts, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
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

// code_source_path is emitted as the source dir relative to the bundle and no code
// is copied — the deploy-time mutator packages it in place. writeBundle produces
// only databricks.yml + launch artifacts, not a code_source copy.
func TestConvertToDabsDoesNotCopyCode(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "train.py"), []byte("print()\n"), 0o600))

	cfg := "experiment_name: wt\ncommand: cd \"$CODE_SOURCE_PATH\" && python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: ./src\n"
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	written, err := writeBundle(t.Context(), loaded, path, dir, false)
	require.NoError(t, err)

	// code_source_path points at the existing source dir; nothing is copied.
	root, _, err := convertToDabs(t.Context(), loaded, path, dir)
	require.NoError(t, err)
	art := "resources.jobs." + loaded.ExperimentName + ".tasks[0].ai_runtime_task"
	assert.Equal(t, "./src", get(t, root, art+".code_source_path").MustString())
	assert.NotContains(t, written, "code_source/")
}

// A code_source root_path outside the bundle directory is rejected: the mutator only
// packages a directory inside the bundle sync root.
func TestConvertToDabsRejectsCodeOutsideBundle(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(outside, "src"), 0o700))

	cfg := "experiment_name: outside\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: " + filepath.Join(outside, "src") + "\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	// Bundle dir is the config's temp dir; the source is in a different tree.
	_, _, err = convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.ErrorContains(t, err, "not inside the bundle")
}

// A git-pinned code_source is emitted as a `tgz` artifact (the DABs artifact
// snapshotter): DABs builds the tarball from the ref at deploy and code_source_path
// points at it.
func TestConvertToDabsGitPinEmitsArtifact(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o700))
	cfg := "experiment_name: gitpin\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: ./src\n    git:\n      commit: abc123\n"
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, dir)
	require.NoError(t, err)

	a := "artifacts." + codeSourceArtifactKey
	assert.Equal(t, "tgz", get(t, root, a+".type").MustString())
	// path is the code dir's parent; include is the basename, so archive entries are
	// "src/..." (the layout the runtime extracts to /databricks/code_source/src).
	assert.Equal(t, ".", get(t, root, a+".path").MustString())
	assert.Equal(t, "abc123", get(t, root, a+".git.commit").MustString())
	inc := get(t, root, a+".include").MustSequence()
	require.Len(t, inc, 1)
	assert.Equal(t, "src", inc[0].MustString())
	assert.Equal(t, "./dist/code_source.tgz", get(t, root, a+".files[0].source").MustString())

	task := "resources.jobs." + loaded.ExperimentName + ".tasks[0].ai_runtime_task"
	assert.Equal(t, "./dist/code_source.tgz", get(t, root, task+".code_source_path").MustString())
}

// A requirements-FILE dependency set (environment.dependencies is a path) is folded
// into the environments[] spec so the deploy-time aicode mutator can regenerate
// requirements.yaml from it. Convert emits no requirements.yaml artifact of its own.
// usage_policy_id is a resolved budget policy id and maps to the job's
// budget_policy_id (usage_policy_name, which needs resolution, is rejected).
func TestConvertToDabsMapsUsagePolicyID(t *testing.T) {
	cfg := minimalConfig + "usage_policy_id: 12345678-90ab-cdef-1234-567890abcdef\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, "12345678-90ab-cdef-1234-567890abcdef", get(t, root, "resources.jobs."+loaded.ExperimentName+".budget_policy_id").MustString())
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

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)

	perms := get(t, root, "resources.jobs."+loaded.ExperimentName+".permissions").MustSequence()
	require.Len(t, perms, 2)
	assert.Equal(t, "CAN_MANAGE", perms[0].Get("level").MustString())
	assert.Equal(t, "alice@example.com", perms[0].Get("user_name").MustString())
	assert.Equal(t, "eng", perms[1].Get("group_name").MustString())
}

// The job resource key is the experiment name as-is. A name that YAML would type
// as a non-string scalar ("12345" -> !!int, "true" -> !!bool) must be emitted
// quoted, or the bundle loader rejects the key with "invalid key tag".
func TestConvertToDabsSafeJobKey(t *testing.T) {
	for _, name := range []string{"12345", "1.5e3", "true", "null", "my-run_1"} {
		assert.Equal(t, name, bundleResourceKey(name), "key for %q", name)
	}

	for _, name := range []string{"12345", "true", "my-run_1"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := "experiment_name: \"" + name + "\"\ncommand: python t.py\n" +
				"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n"
			path := filepath.Join(dir, "run.yaml")
			require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
			loaded, err := loadRunConfig(path)
			require.NoError(t, err)

			root, _, err := convertToDabs(t.Context(), loaded, path, dir)
			require.NoError(t, err)
			// The key is the raw name; name/experiment keep the same value.
			jobs, err := dyn.GetByPath(dyn.V(root), dyn.MustPathFromString("resources.jobs"))
			require.NoError(t, err)
			job := jobs.Get(name)
			require.True(t, job.IsValid(), "job must be keyed by %q", name)
			assert.Equal(t, name, job.Get("name").MustString())

			// The emitted YAML must load back with the key still a string.
			_, err = writeBundle(t.Context(), loaded, path, dir, true)
			require.NoError(t, err)
			emitted, err := os.ReadFile(filepath.Join(dir, "databricks.yml"))
			require.NoError(t, err)
			var doc struct {
				Resources struct {
					Jobs map[string]struct {
						Name string `yaml:"name"`
					} `yaml:"jobs"`
				} `yaml:"resources"`
			}
			require.NoError(t, yaml.Unmarshal(emitted, &doc), "emitted YAML must parse:\n%s", emitted)
			require.Contains(t, doc.Resources.Jobs, name, "job key must load as the string %q:\n%s", name, emitted)
		})
	}
}

// include_paths narrows the archive to a subset of root_path: emitted as the `tgz`
// artifact's include (relative to the code-source root), matching air CLI semantics.
func TestConvertToDabsIncludePathsEmitsArtifact(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o700))
	cfg := "experiment_name: inc\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: ./src\n" +
		"    include_paths:\n      - keep\n"
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, dir)
	require.NoError(t, err)

	a := "artifacts." + codeSourceArtifactKey
	assert.Equal(t, "tgz", get(t, root, a+".type").MustString())
	// path is the code dir's parent; include_paths are basename-prefixed, so the
	// entry is "src/keep" (relative to the bundle root, path ".").
	assert.Equal(t, ".", get(t, root, a+".path").MustString())
	inc := get(t, root, a+".include").MustSequence()
	require.Len(t, inc, 1)
	assert.Equal(t, "src/keep", inc[0].MustString())

	task := "resources.jobs." + loaded.ExperimentName + ".tasks[0].ai_runtime_task"
	assert.Equal(t, "./dist/code_source.tgz", get(t, root, task+".code_source_path").MustString())
}

// include_paths pointing at directories (the primary use case, not single files):
// each entry is basename-prefixed the same way, so a subdirectory `pkg` under a
// `./src` root becomes `src/pkg`. Convert is a syntactic mapping — whether the entry
// names a directory or a file is identical here; the tgz builder (#6428) walks a
// directory entry recursively at deploy.
func TestConvertToDabsIncludePathsDirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src", "pkg"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src", "config"), 0o700))
	cfg := "experiment_name: incdir\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: ./src\n" +
		"    include_paths:\n      - pkg\n      - config\n"
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, dir)
	require.NoError(t, err)

	a := "artifacts." + codeSourceArtifactKey
	assert.Equal(t, ".", get(t, root, a+".path").MustString())
	inc := get(t, root, a+".include").MustSequence()
	require.Len(t, inc, 2)
	assert.Equal(t, "src/pkg", inc[0].MustString())
	assert.Equal(t, "src/config", inc[1].MustString())
}

// root_path "." (the code source is the whole bundle directory) has no basename to
// nest the archive under, and an include rooted at "." would sweep the bundle's own
// generated files into the tarball — so a git/include snapshot there is rejected with
// a message pointing at a subdirectory. (A plain snapshot with root_path "." is not an
// artifact case and is unaffected.)
func TestConvertToDabsDotRootPathRejected(t *testing.T) {
	for _, tc := range []struct{ name, snap string }{
		{"git", "code_source:\n  type: snapshot\n  snapshot:\n    root_path: .\n    git:\n      commit: abc123\n"},
		{"include", "code_source:\n  type: snapshot\n  snapshot:\n    root_path: .\n    include_paths:\n      - foo\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "foo"), 0o700))
			cfg := "experiment_name: dot\ncommand: python train.py\n" +
				"compute: {accelerator_type: GPU_1xA10, num_accelerators: 1}\n" + tc.snap
			path := filepath.Join(dir, "run.yaml")
			require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
			loaded, err := loadRunConfig(path)
			require.NoError(t, err)

			_, _, err = convertToDabs(t.Context(), loaded, path, dir)
			require.ErrorContains(t, err, "resolves to the bundle root")
		})
	}
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
		_, _, err = convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
		require.ErrorContains(t, err, "docker_image is not yet supported")
	})

	t.Run("usage_policy_name", func(t *testing.T) {
		cfg := minimalConfig + "usage_policy_name: my-policy\n"
		path := writeConfigFile(t, "run.yaml", cfg)
		loaded, err := loadRunConfig(path)
		require.NoError(t, err)
		_, _, err = convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
		require.ErrorContains(t, err, "usage_policy_name is not yet supported")
	})
}
