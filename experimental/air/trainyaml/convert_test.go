package trainyaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTrainYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "train.yaml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestParseAndConvertSnapshot(t *testing.T) {
	p := writeTrainYAML(t, `
experiment_name: my-training
compute:
  num_accelerators: 8
  accelerator_type: GPU_8xH100
environment:
  version: "5"
  dependencies:
    - torch>=2.0.0
code_source:
  type: snapshot
  snapshot:
    root_path: ./src
command: |
  cd $CODE_SOURCE_PATH
  python train.py
max_retries: 2
timeout_minutes: 30
`)

	cfg, err := Parse(p)
	require.NoError(t, err)

	res, err := Convert(cfg)
	require.NoError(t, err)
	require.NotNil(t, res.Job)

	task := res.Job.Tasks[0]
	assert.Equal(t, "my-training", task.TaskKey)
	assert.Equal(t, "default", task.EnvironmentKey)
	assert.Equal(t, 2, task.MaxRetries)
	assert.Equal(t, 1800, task.TimeoutSeconds)

	air := task.AiRuntimeTask
	require.NotNil(t, air)
	assert.Equal(t, "my-training", air.Experiment)
	assert.Equal(t, "./src", air.CodeSourcePath)
	require.Len(t, air.Deployments, 1)
	assert.Equal(t, "src/command.sh", air.Deployments[0].CommandPath)
	assert.EqualValues(t, "GPU_8xH100", air.Deployments[0].Compute.AcceleratorType)
	assert.Equal(t, 8, air.Deployments[0].Compute.AcceleratorCount)

	require.Len(t, res.Job.Environments, 1)
	assert.Equal(t, "default", res.Job.Environments[0].EnvironmentKey)
	assert.Equal(t, "5", res.Job.Environments[0].Spec.EnvironmentVersion)
	assert.Equal(t, []string{"torch>=2.0.0"}, res.Job.Environments[0].Spec.Dependencies)

	assert.Contains(t, res.CommandScript, "python train.py")
	assert.Empty(t, res.ArtifactPath)
}

func TestConvertRemoteVolumeSetsArtifactPath(t *testing.T) {
	cfg := &Config{
		ExperimentName: "exp",
		Command:        "echo hi",
		Compute:        Compute{NumAccelerators: 1, AcceleratorType: "GPU_1xA10"},
		CodeSource: &CodeSource{
			Type:     "snapshot",
			Snapshot: &Snapshot{RootPath: "code", RemoteVolume: "/Volumes/main/default/code"},
		},
	}
	res, err := Convert(cfg)
	require.NoError(t, err)
	assert.Equal(t, "/Volumes/main/default/code", res.ArtifactPath)
}

func TestConvertRejectsLegacyAccelerator(t *testing.T) {
	cfg := &Config{
		ExperimentName: "exp",
		Command:        "echo hi",
		Compute:        Compute{NumAccelerators: 1, AcceleratorType: "a10"},
		CodeSource:     &CodeSource{Type: "snapshot", Snapshot: &Snapshot{RootPath: "code"}},
	}
	_, err := Convert(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported by ai_runtime_task")
}

func TestConvertRejectsDockerImage(t *testing.T) {
	cfg := &Config{
		ExperimentName: "exp",
		Command:        "echo hi",
		Compute:        Compute{NumAccelerators: 1, AcceleratorType: "GPU_1xA10"},
		Environment:    &Environment{DockerImage: &DockerImage{URL: "org/repo:tag"}},
		CodeSource:     &CodeSource{Type: "snapshot", Snapshot: &Snapshot{RootPath: "code"}},
	}
	_, err := Convert(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker_image is not supported")
}

func TestConvertWarnsOnUnmappedFields(t *testing.T) {
	cfg := &Config{
		ExperimentName: "exp",
		Command:        "echo hi",
		Compute:        Compute{NumAccelerators: 1, AcceleratorType: "GPU_1xA10"},
		Secrets:        map[string]string{"HF_TOKEN": "scope/key"},
		CodeSource: &CodeSource{
			Type:     "snapshot",
			Snapshot: &Snapshot{RootPath: "code", IncludePaths: []string{"a", "b"}},
		},
	}
	res, err := Convert(cfg)
	require.NoError(t, err)
	require.Len(t, res.Warnings, 2)
	assert.Contains(t, res.Warnings[0], "secrets")
	assert.Contains(t, res.Warnings[1], "include_paths")
}

func TestParseRejectsUnknownField(t *testing.T) {
	p := writeTrainYAML(t, "experiment_name: x\nbogus_field: 1\n")
	_, err := Parse(p)
	require.Error(t, err)
}
