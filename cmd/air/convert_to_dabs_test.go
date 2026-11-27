package aircmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
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

// sharedCodeWorkload writes a workload YAML in dir that points at dir/src, so two
// workloads can be converted into the same directory.
func sharedCodeWorkload(t *testing.T, dir, name, extra string) (*runConfig, string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o700))
	cfg := "experiment_name: " + name + "\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: ./src\n" + extra
	path := filepath.Join(dir, name+".yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)
	return loaded, path
}

const sidecarsConfig = "env_variables: {FOO: bar}\nsecrets: {TOKEN: scope/key}\nparameters: {lr: 0.1}\n"

// --force re-converts the same workload only: a databricks.yml that names a different
// bundle is left alone, together with its sidecars, and the error names both bundles.
func TestConvertToDabsForceRefusesDifferentBundle(t *testing.T) {
	dir := t.TempDir()
	cfgA, pathA := sharedCodeWorkload(t, dir, "exp-a", sidecarsConfig)
	_, err := writeBundle(t.Context(), cfgA, pathA, dir, false)
	require.NoError(t, err)
	bundleA := testutil.ReadFile(t, filepath.Join(dir, "databricks.yml"))

	cfgB, pathB := sharedCodeWorkload(t, dir, "exp-b", "")
	want := fmt.Sprintf("databricks.yml in %s belongs to bundle \"exp-a\", not to this workload's bundle \"exp-b\"; pass --output-dir with another directory that contains the code directory", dir)
	_, err = writeBundle(t.Context(), cfgB, pathB, dir, true)
	require.EqualError(t, err, want)
	assert.Equal(t, bundleA, testutil.ReadFile(t, filepath.Join(dir, "databricks.yml")))
	assert.FileExists(t, filepath.Join(dir, generatedArtifactsDir, envVarsName))
	assert.FileExists(t, filepath.Join(dir, generatedArtifactsDir, secretEnvVarsName))
	assert.FileExists(t, filepath.Join(dir, generatedArtifactsDir, hyperparametersName))

	// Without --force the same error applies: suggesting --force would be a dead end.
	_, err = writeBundle(t.Context(), cfgB, pathB, dir, false)
	require.EqualError(t, err, want)
}

// Re-converting the same workload with --force removes the sidecars the previous
// conversion wrote and this one does not, so sync.paths does not upload stale ones.
func TestConvertToDabsForceRemovesStaleSidecars(t *testing.T) {
	dir := t.TempDir()
	cfg, path := sharedCodeWorkload(t, dir, "exp-a", sidecarsConfig)
	_, err := writeBundle(t.Context(), cfg, path, dir, false)
	require.NoError(t, err)

	cfg, path = sharedCodeWorkload(t, dir, "exp-a", "")
	_, err = writeBundle(t.Context(), cfg, path, dir, true)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, generatedArtifactsDir, commandScriptName))
	assert.NoFileExists(t, filepath.Join(dir, generatedArtifactsDir, envVarsName))
	assert.NoFileExists(t, filepath.Join(dir, generatedArtifactsDir, secretEnvVarsName))
	assert.NoFileExists(t, filepath.Join(dir, generatedArtifactsDir, hyperparametersName))
}

func TestConvertToDabsExistingBundleRoot(t *testing.T) {
	for _, name := range config.FileNames {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFile(t, "run.yaml", minimalConfig+"code_source:\n  root_path: ./src\n")
			dir := filepath.Dir(path)
			require.NoError(t, os.Mkdir(filepath.Join(dir, "src"), 0o700))
			loaded, err := loadRunConfig(path)
			require.NoError(t, err)
			existing := filepath.Join(dir, name)
			edited := []byte("# hand-edited\n")
			require.NoError(t, os.WriteFile(existing, edited, 0o600))

			_, err = writeBundle(t.Context(), loaded, path, dir, false)
			want := fmt.Sprintf("%s already exists in %s; move or remove it before converting", name, dir)
			if name == "databricks.yml" {
				want = fmt.Sprintf("databricks.yml already exists in %s; pass --force to overwrite or remove it", dir)
			}
			require.EqualError(t, err, want)
			found, err := config.FileNames.FindInPath(dir)
			require.NoError(t, err)
			assert.Equal(t, existing, found)
			kept, err := os.ReadFile(existing)
			require.NoError(t, err)
			assert.Equal(t, edited, kept)
		})
	}
}

func TestConvertToDabsRejectsAlternateRoot(t *testing.T) {
	const canonical = "databricks.yml"
	script := filepath.Join(generatedArtifactsDir, commandScriptName)
	for _, alias := range config.FileNames {
		if alias == canonical {
			continue
		}
		for _, tc := range []struct {
			name  string
			force bool
			files []string
		}{
			{"force", true, []string{alias}},
			{"force-with-canonical", true, []string{canonical, alias, script}},
			{"with-canonical", false, []string{canonical, alias}},
		} {
			t.Run(alias+"/"+tc.name, func(t *testing.T) {
				dir := t.TempDir()
				cfg, path := rayLayoutConfig(t, dir, "core")
				for _, name := range tc.files {
					testutil.WriteFile(t, filepath.Join(dir, name), name)
				}

				_, err := writeBundle(t.Context(), cfg, path, dir, tc.force)
				assert.EqualError(t, err, fmt.Sprintf("%s already exists in %s; move or remove it before converting", alias, dir))
				for _, name := range tc.files {
					assert.Equal(t, name, testutil.ReadFile(t, filepath.Join(dir, name)))
				}
				if !slices.Contains(tc.files, canonical) {
					assert.NoFileExists(t, filepath.Join(dir, canonical))
				}
				if !slices.Contains(tc.files, script) {
					assert.NoDirExists(t, filepath.Join(dir, generatedArtifactsDir))
				}
			})
		}
	}
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
	assert.Equal(t, "convert-to-dabs YAML_PATH", cmd.Use)
	assert.Empty(t, cmd.Commands(), "convert-to-dabs must not register subcommands")
	// Exactly one positional (the YAML path).
	assert.NoError(t, cmd.Args(cmd, []string{"run.yaml"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"a", "b"}))
}

// val is a value of the emitted bundle root.
type val struct{ v any }

func (v val) MustString() string { return v.v.(string) }
func (v val) MustInt() int64     { return int64(v.v.(int)) }
func (v val) MustBool() bool     { return v.v.(bool) }

func (v val) MustSequence() []val {
	if ss, ok := v.v.([]string); ok {
		out := make([]val, len(ss))
		for i, s := range ss {
			out[i] = val{s}
		}
		return out
	}
	var out []val
	for _, e := range v.v.([]any) {
		out = append(out, val{e})
	}
	return out
}

func (v val) Get(key string) val {
	for _, p := range v.v.(structyaml.Map) {
		if p.Key == key {
			return val{p.Value}
		}
	}
	return val{}
}

// lookup reads a dotted path, with optional [index] suffixes, out of the emitted bundle root.
func lookup(root structyaml.Map, path string) (val, bool) {
	v := val{root}
	for part := range strings.SplitSeq(path, ".") {
		key, idx, hasIdx := strings.Cut(strings.TrimSuffix(part, "]"), "[")
		if _, ok := v.v.(structyaml.Map); !ok {
			return val{}, false
		}
		if v = v.Get(key); v.v == nil {
			return val{}, false
		}
		if hasIdx {
			i, err := strconv.Atoi(idx)
			seq, ok := v.v.([]any)
			if err != nil || !ok || i >= len(seq) {
				return val{}, false
			}
			v = val{seq[i]}
		}
	}
	return v, true
}

// get is a small helper: read a dotted path out of the emitted bundle root.
func get(t *testing.T, root structyaml.Map, path string) val {
	t.Helper()
	v, ok := lookup(root, path)
	require.True(t, ok, "path %q should exist", path)
	return v
}

func has(root structyaml.Map, path string) bool {
	_, ok := lookup(root, path)
	return ok
}

// A full config maps onto a schema-shaped bundle: bundle name, job/task keys, the
// ai_runtime_task (experiment + single deployment + code_source_path), framework
// fields on the task wrapper, and the environment spec.
func TestConvertToDabsFullMapping(t *testing.T) {
	cfg := `
experiment_name: my-run
command: python train.py
compute:
  accelerator_type: GPU_1xH100
  num_accelerators: 1
  accelerator_pool_id: capacity-1
max_retries: 2
timeout_minutes: 30
mlflow_run_name: run-42
mlflow_artifact_location: dbfs:/Volumes/main/default/artifacts
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
	assert.True(t, get(t, root, task+".retry_on_timeout").MustBool())
	assert.Equal(t, int64(1800), get(t, root, task+".timeout_seconds").MustInt())
	assert.False(t, has(root, task+".ai_runtime_task.max_retries"), "retries must not be inside ai_runtime_task")

	art := task + ".ai_runtime_task"
	assert.Equal(t, name, get(t, root, art+".experiment").MustString())
	assert.Equal(t, "run-42", get(t, root, art+".mlflow_run").MustString())
	assert.Equal(t, "dbfs:/Volumes/main/default/artifacts", get(t, root, art+".mlflow_artifact_location").MustString())
	// code_source is packaged as a `tgz` artifact; code_source_path points at the
	// built tarball, and the artifact packs the code dir (path = its parent, include =
	// its basename, so entries nest under "src/").
	assert.Equal(t, "./dist/code_source.tgz", get(t, root, art+".code_source_path").MustString())
	codeArt := "artifacts." + codeSourceArtifactKey
	assert.Equal(t, "tgz", get(t, root, codeArt+".type").MustString())
	assert.Equal(t, ".", get(t, root, codeArt+".path").MustString())
	codeInc := get(t, root, codeArt+".include").MustSequence()
	require.Len(t, codeInc, 1)
	assert.Equal(t, "src", codeInc[0].MustString())
	assert.Equal(t, "./dist/code_source.tgz", get(t, root, codeArt+".files[0].source").MustString())

	dep := art + ".deployments[0]"
	assert.Equal(t, "./"+generatedArtifactsDir+"/"+commandScriptName, get(t, root, dep+".command_path").MustString())
	assert.Equal(t, "GPU_1xH100", get(t, root, dep+".compute.accelerator_type").MustString())
	assert.Equal(t, int64(1), get(t, root, dep+".compute.accelerator_count").MustInt())
	assert.Equal(t, "capacity-1", get(t, root, dep+".compute.provisioned_capacity_id").MustString())

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

func TestConvertToDabsPreservesPoolPriorityAndUnityCatalogImage(t *testing.T) {
	for _, tc := range []struct {
		name          string
		poolID        string
		priorityClass string
		imagePath     string
	}{
		{name: "unset"},
		{name: "pool", poolID: "capacity-1"},
		{name: "pool-priority", poolID: "capacity-1", priorityClass: "NORMAL"},
		{name: "image", imagePath: "main.air.training:prod"},
		{name: "pool-image", poolID: "capacity-1", imagePath: "main.air.training:prod"},
		{name: "pool-priority-image", poolID: "capacity-1", priorityClass: "NORMAL", imagePath: "main.air.training:prod"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := minimalConfig
			var computeOptions string
			if tc.poolID != "" {
				computeOptions += "  accelerator_pool_id: " + tc.poolID + "\n"
			}
			if tc.priorityClass != "" {
				computeOptions += "  priority_class: " + tc.priorityClass + "\n"
			}
			cfg = strings.Replace(cfg, "  num_accelerators: 1\n", "  num_accelerators: 1\n"+computeOptions, 1)
			if tc.imagePath != "" {
				cfg += "environment:\n  unity_catalog_image: " + tc.imagePath + "\n"
			}
			path := writeConfigFile(t, "run.yaml", cfg)
			loaded, err := loadRunConfig(path)
			require.NoError(t, err)

			root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
			require.NoError(t, err)

			aiRuntimeTask := "resources.jobs." + loaded.ExperimentName + ".tasks[0].ai_runtime_task"
			compute := aiRuntimeTask + ".deployments[0].compute"
			if tc.poolID == "" {
				assert.False(t, has(root, compute+".provisioned_capacity_id"))
			} else {
				assert.Equal(t, tc.poolID, get(t, root, compute+".provisioned_capacity_id").MustString())
			}
			if tc.priorityClass == "" {
				assert.False(t, has(root, aiRuntimeTask+".priority_class"))
			} else {
				assert.Equal(t, tc.priorityClass, get(t, root, aiRuntimeTask+".priority_class").MustString())
			}
			if tc.imagePath == "" {
				assert.False(t, has(root, aiRuntimeTask+".unity_catalog_image_path"))
			} else {
				assert.Equal(t, tc.imagePath, get(t, root, aiRuntimeTask+".unity_catalog_image_path").MustString())
			}
		})
	}
}

// Optional fields are omitted rather than emitted empty: no code_source means no
// code_source_path, and an unset timeout means no timeout_seconds.
// sync.paths lists only the generated-artifacts dir. The code directory must be
// absent: the `tgz` artifact packages it, so syncing it as loose files too would
// upload the whole tree a second time.
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

	// The task points at the built `tgz` artifact; the code dir is packaged through the
	// artifact path, so the sync set omits it.
	task := "resources.jobs." + loaded.ExperimentName + ".tasks[0].ai_runtime_task"
	assert.Equal(t, "./dist/code_source.tgz", get(t, root, task+".code_source_path").MustString())
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
	assert.True(t, get(t, root, task+".retry_on_timeout").MustBool())
	assert.False(t, has(root, task+".timeout_seconds"))
	assert.False(t, has(root, task+".ai_runtime_task.code_source_path"))
	assert.False(t, has(root, task+".ai_runtime_task.mlflow_run"))
	// The command still needs a home even without code_source.
	assert.Equal(t, "./"+generatedArtifactsDir+"/"+commandScriptName, get(t, root, task+".ai_runtime_task.deployments[0].command_path").MustString())

	// Even with no environment block, the default runtime version is pinned (what
	// `air run` would have used) rather than emitting an empty environment spec.
	env := "resources.jobs." + name + ".environments[0]"
	assert.Equal(t, "6", get(t, root, env+".spec.environment_version").MustString())
	assert.False(t, has(root, env+".spec.dependencies"))
}

func TestConvertToDabsDisablesTimeoutRetriesWhenRetriesAreDisabled(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", minimalConfig+"max_retries: 0\n")
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)

	task := "resources.jobs." + loaded.ExperimentName + ".tasks[0]"
	assert.Equal(t, int64(0), get(t, root, task+".max_retries").MustInt())
	assert.False(t, get(t, root, task+".retry_on_timeout").MustBool())
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

func TestConvertToDabsSupportsB300(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", `
experiment_name: b300
command: python train.py
compute:
  accelerator_type: GPU_8xB300
  num_accelerators: 16
environment:
  version: 6
  dependencies: []
`)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)

	deployment := "resources.jobs.b300.tasks[0].ai_runtime_task.deployments[0]"
	assert.Equal(t, "GPU_8xB300", get(t, root, deployment+".compute.accelerator_type").MustString())
	assert.Equal(t, int64(16), get(t, root, deployment+".compute.accelerator_count").MustInt())
	assert.Equal(t, "6", get(t, root, "resources.jobs.b300.environments[0].spec.environment_version").MustString())
}

func TestConvertToDabsDatabricksAIEnvironment(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", minimalConfig+"\nenvironment:\n  version: databricks_ai_v5\n")
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.NoError(t, err)

	spec := "resources.jobs." + loaded.ExperimentName + ".environments[0].spec."
	assert.Equal(t, "workspace-base-environments/databricks_ai_v5", get(t, root, spec+"base_environment").MustString())
	assert.False(t, has(root, spec+"environment_version"))
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
	require.ErrorContains(t, err, "remote_volume in code_source is not supported")
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

// code_source_path points at the built `tgz` artifact and no code is copied — the
// artifact snapshotter packages the source dir at deploy. writeBundle produces only
// databricks.yml + launch artifacts, not a code_source copy.
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

	// code_source_path points at the built tarball; nothing is copied.
	root, _, err := convertToDabs(t.Context(), loaded, path, dir)
	require.NoError(t, err)
	art := "resources.jobs." + loaded.ExperimentName + ".tasks[0].ai_runtime_task"
	assert.Equal(t, "./dist/code_source.tgz", get(t, root, art+".code_source_path").MustString())
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
// into the environments[] spec, which the runtime installs deps from directly.
// Convert emits no requirements.yaml artifact of its own.
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
			assert.Equal(t, name, get(t, root, "resources.jobs."+name+".name").MustString())

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
// generated files into the tarball, so every snapshot there is rejected. A working-tree
// snapshot is offered --output-dir; a git snapshot is archived from the bundle root, so
// it is offered a parent only when the bundle directory is below the repository's top
// level (the parent is then still inside the repository).
func TestConvertToDabsDotRootPathRejected(t *testing.T) {
	gitSnap := "code_source:\n  type: snapshot\n  snapshot:\n    root_path: .\n    git:\n      commit: abc123\n"
	for _, tc := range []struct {
		name, snap        string
		repoSubdir        bool // the bundle dir is a subdirectory of a git repository
		wantOutputDirHint bool
	}{
		{"plain", "code_source:\n  type: snapshot\n  snapshot:\n    root_path: .\n", false, true},
		{"include", "code_source:\n  type: snapshot\n  snapshot:\n    root_path: .\n    include_paths:\n      - foo\n", false, true},
		{"git-no-repo", gitSnap, false, false},
		{"git-repo-subdir", gitSnap, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.repoSubdir {
				runGit(t, dir, "init", "-q")
				dir = filepath.Join(dir, "proj")
			}
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "foo"), 0o700))
			cfg := "experiment_name: dot\ncommand: python train.py\n" +
				"compute: {accelerator_type: GPU_1xA10, num_accelerators: 1}\n" + tc.snap
			path := filepath.Join(dir, "run.yaml")
			require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
			loaded, err := loadRunConfig(path)
			require.NoError(t, err)

			_, _, err = convertToDabs(t.Context(), loaded, path, dir)
			require.ErrorContains(t, err, "resolves to the bundle root")
			require.ErrorContains(t, err, "subdirectory")
			if tc.wantOutputDirHint {
				require.ErrorContains(t, err, "--output-dir")
			} else {
				require.NotContains(t, err.Error(), "--output-dir")
			}
		})
	}
}

// The --output-dir remedy the rejection suggests: with the bundle rooted at the
// parent, the code dir has a basename to nest under and converts.
func TestConvertToDabsDotRootPathWithParentOutputDir(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "proj")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	cfg := "experiment_name: dot\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xA10, num_accelerators: 1}\n" +
		"code_source:\n  type: snapshot\n  snapshot:\n    root_path: .\n"
	path := filepath.Join(dir, "run.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	root, _, err := convertToDabs(t.Context(), loaded, path, parent)
	require.NoError(t, err)

	a := "artifacts." + codeSourceArtifactKey
	assert.Equal(t, ".", get(t, root, a+".path").MustString())
	inc := get(t, root, a+".include").MustSequence()
	require.Len(t, inc, 1)
	assert.Equal(t, "proj", inc[0].MustString())
}

func TestRetryDefaultsNote(t *testing.T) {
	note := retryDefaultsNote(&runConfig{})
	assert.Contains(t, note, "max_retries: 3")
	assert.Contains(t, note, "retry_on_timeout: true")

	// An explicit value, including 0, is the user's own: nothing to explain.
	assert.Empty(t, retryDefaultsNote(&runConfig{MaxRetries: new(0)}))
	assert.Empty(t, retryDefaultsNote(&runConfig{MaxRetries: new(2)}))
}

func TestConvertToDabsCommandPrintsRetryNoteOnlyWhenUnset(t *testing.T) {
	for _, tc := range []struct {
		name, extra string
		wantNote    bool
	}{
		{"unset", "", true},
		{"zero", "max_retries: 0\n", false},
		{"set", "max_retries: 2\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfigFile(t, "run.yaml", minimalConfig+tc.extra)
			ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
			cmd := newConvertToDabsCommand()
			cmd.SetContext(ctx)
			require.NoError(t, cmd.RunE(cmd, []string{path}))

			assert.Contains(t, stderr.String(), "Wrote a Databricks Asset Bundle")
			assert.Equal(t, tc.wantNote, strings.Contains(stderr.String(), "does not set max_retries"))
		})
	}
}

func TestConvertToDabsRejectsUnsupported(t *testing.T) {
	t.Run("usage_policy_name", func(t *testing.T) {
		cfg := minimalConfig + "usage_policy_name: my-policy\n"
		path := writeConfigFile(t, "run.yaml", cfg)
		loaded, err := loadRunConfig(path)
		require.NoError(t, err)
		_, _, err = convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
		require.ErrorContains(t, err, "usage_policy_name is not yet supported")
	})
}

// rayLayoutConfig writes ray/<recipe>/workload.yaml under dir, snapshotting its parent
// (root_path: ..) to include the shared ray/ray_bootstrap.sh, and loads it.
func rayLayoutConfig(t *testing.T, dir, recipe string) (*runConfig, string) {
	t.Helper()
	recipeDir := filepath.Join(dir, "ray", recipe)
	require.NoError(t, os.MkdirAll(recipeDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ray", "ray_bootstrap.sh"), []byte("#!/bin/bash\n"), 0o600))
	cfg := "experiment_name: ray-" + recipe + "\ncommand: bash ray_bootstrap.sh\n" +
		"compute: {accelerator_type: GPU_1xA10, num_accelerators: 2}\n" +
		"code_source:\n  root_path: ..\n  include_paths: [ray_bootstrap.sh, " + recipe + "]\n"
	path := filepath.Join(recipeDir, "workload.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)
	return loaded, path
}

// A workload whose root_path is its parent directory gets an --output-dir that
// converts it, not advice to edit a databricks.yml that was never written.
func TestConvertToDabsCodeOutsideBundleSuggestsOutputDir(t *testing.T) {
	dir := t.TempDir()
	loaded, path := rayLayoutConfig(t, dir, "core")

	_, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.ErrorContains(t, err, "not inside the bundle")
	assert.Contains(t, err.Error(), "("+quotedPath(filepath.Join(dir, "ray"))+")")
	assert.Contains(t, err.Error(), "convert-to-dabs can package only code inside the bundle directory")
	quoted, ok := shellquote.Arg(runtime.GOOS, dir)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "--output-dir "+quoted+" ")
	assert.Contains(t, err.Error(), "or move the code inside the bundle directory")
	assert.NotContains(t, err.Error(), "sync.paths")

	root, _, err := convertToDabs(t.Context(), loaded, path, dir)
	require.NoError(t, err)
	assert.Equal(t, ".", get(t, root, "artifacts."+codeSourceArtifactKey+".path").MustString())
	inc := get(t, root, "artifacts."+codeSourceArtifactKey+".include").MustSequence()
	require.Len(t, inc, 2)
	assert.Equal(t, "ray/ray_bootstrap.sh", inc[0].MustString())
	assert.Equal(t, "ray/core", inc[1].MustString())
}

// Sibling workloads that share a code directory would all be sent to its parent, and
// converting a second one there would replace the first bundle. Once the parent holds
// a bundle, under any of the names DABs accepts, it is not suggested.
func TestConvertToDabsCodeOutsideBundleSkipsParentWithBundle(t *testing.T) {
	for _, name := range config.FileNames {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			rayLayoutConfig(t, dir, "core")
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("bundle:\n  name: ray-core\n"), 0o600))
			loaded, path := rayLayoutConfig(t, dir, "data")

			_, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
			require.ErrorContains(t, err, "not inside the bundle")
			assert.Contains(t, err.Error(), fmt.Sprintf("%s already holds a bundle (%s)", quotedPath(dir), name))
			assert.NotContains(t, err.Error(), "to write the bundle to the code directory's parent")
		})
	}
}

// The suggested directory is quoted for the user's shell, so a parent whose path
// has a space pastes as one argument.
func TestConvertToDabsCodeOutsideBundleQuotesOutputDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my recipes")
	loaded, path := rayLayoutConfig(t, dir, "core")

	_, _, err := convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.ErrorContains(t, err, "not inside the bundle")
	quoted, ok := shellquote.Arg(runtime.GOOS, dir)
	require.True(t, ok)
	require.NotEqual(t, dir, quoted)
	assert.Contains(t, err.Error(), "--output-dir "+quoted+" ")
}

// Paths in the error keep their native separators, single backslashes on Windows,
// and fall back to Go quoting when the shell can't quote them.
func TestQuotedPath(t *testing.T) {
	plain := filepath.Join("work", "ray")
	assert.Equal(t, plain, quotedPath(plain))
	assert.Equal(t, `"bad\nname"`, quotedPath("bad\nname"))
}

// A git-pinned snapshot is archived from the bundle root. When the code directory is
// the repository's top level, its parent is outside the repository, so no parent is
// offered.
func TestConvertToDabsGitPinOutsideBundleDoesNotSuggestOutputDir(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(outside, "src"), 0o700))
	runGit(t, filepath.Join(outside, "src"), "init", "-q")
	cfg := "experiment_name: outside\ncommand: python train.py\n" +
		"compute: {accelerator_type: GPU_1xH100, num_accelerators: 1}\n" +
		"code_source:\n  root_path: " + filepath.Join(outside, "src") + "\n  git:\n    commit: abc123\n"
	path := writeConfigFile(t, "run.yaml", cfg)
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	_, _, err = convertToDabs(t.Context(), loaded, path, filepath.Dir(path))
	require.ErrorContains(t, err, "not inside the bundle")
	assert.Contains(t, err.Error(), "; move the code inside the bundle directory")
	assert.NotContains(t, err.Error(), "--output-dir")
}

// A git-pinned code directory below the repository's top level (ray/ in a repo, with
// the workload in ray/core) has a parent inside the repository, so that parent is
// offered as --output-dir, and converting there emits the git-pinned artifact.
func TestConvertToDabsGitPinOutsideBundleSuggestsOutputDirInRepoSubdir(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	recipeDir := filepath.Join(dir, "ray", "core")
	require.NoError(t, os.MkdirAll(recipeDir, 0o700))
	cfg := "experiment_name: ray-core\ncommand: bash ray_bootstrap.sh\n" +
		"compute: {accelerator_type: GPU_1xA10, num_accelerators: 2}\n" +
		"code_source:\n  root_path: ..\n  git:\n    commit: abc123\n"
	path := filepath.Join(recipeDir, "workload.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	loaded, err := loadRunConfig(path)
	require.NoError(t, err)

	_, _, err = convertToDabs(t.Context(), loaded, path, recipeDir)
	require.ErrorContains(t, err, "not inside the bundle")
	quoted, ok := shellquote.Arg(runtime.GOOS, dir)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "--output-dir "+quoted+" ")

	root, _, err := convertToDabs(t.Context(), loaded, path, dir)
	require.NoError(t, err)
	a := "artifacts." + codeSourceArtifactKey
	assert.Equal(t, ".", get(t, root, a+".path").MustString())
	assert.Equal(t, "abc123", get(t, root, a+".git.commit").MustString())
	inc := get(t, root, a+".include").MustSequence()
	require.Len(t, inc, 1)
	assert.Equal(t, "ray", inc[0].MustString())
}

// reconvertSharedCode runs the conversion with the same arguments as a workload author.
func reconvertSharedCode(t *testing.T, path string, flags ...string) error {
	t.Helper()
	cmd := newConvertToDabsCommand()
	ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
	cmd.SetContext(ctx)
	cmd.SetArgs(append(flags, path))
	return cmd.Execute()
}

func TestConvertToDabsReconvertSharedCode(t *testing.T) {
	dir := t.TempDir()
	_, path := rayLayoutConfig(t, dir, "core")

	require.NoError(t, reconvertSharedCode(t, path, "--output-dir", dir))
	quoted, ok := shellquote.Arg(runtime.GOOS, dir)
	require.True(t, ok)
	require.ErrorContains(t, reconvertSharedCode(t, path), "--output-dir "+quoted+" --force")
	require.NoError(t, reconvertSharedCode(t, path, "--output-dir", dir, "--force"))
	for _, content := range []string{"bundle: {name: ray-data}", "bundle: {}", "bundle: ["} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(content), 0o600))
		err := reconvertSharedCode(t, path)
		require.ErrorContains(t, err, "already holds a bundle")
		assert.NotContains(t, err.Error(), "--force")
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yaml"), []byte("bundle: {name: ray-core}"), 0o600))
	for _, duplicate := range []bool{true, false} {
		yml := filepath.Join(dir, "databricks.yml")
		if duplicate {
			require.NoError(t, os.WriteFile(yml, []byte("bundle: {name: ray-core}"), 0o600))
		} else {
			require.NoError(t, os.Remove(yml))
		}
		err := reconvertSharedCode(t, path)
		require.ErrorContains(t, err, "already holds a bundle")
		assert.NotContains(t, err.Error(), "--force")
	}
}
