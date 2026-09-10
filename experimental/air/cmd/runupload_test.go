package aircmd

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// fakeWriter records artifact writes in place of a workspace filer.
type fakeWriter struct {
	written map[string]string
}

func (f *fakeWriter) Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error {
	if f.written == nil {
		f.written = map[string]string{}
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	f.written[name] = string(data)
	return nil
}

func writeConfigFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func itemNames(items []uploadItem) []string {
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.name
	}
	return names
}

func itemData(t *testing.T, items []uploadItem, name string) []byte {
	t.Helper()
	for _, item := range items {
		if item.name == name {
			return item.data
		}
	}
	require.FailNow(t, "artifact not found", name)
	return nil
}

func TestBuildArtifacts_CommandAndConfig(t *testing.T) {
	cfg, err := loadRunConfig(writeConfigFile(t, "run.yaml", minimalConfig))
	require.NoError(t, err)

	items, err := buildArtifacts(cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{trainingConfigName, commandScriptName}, itemNames(items))
	assert.YAMLEq(t, minimalConfig, string(items[0].data))
	assert.Equal(t, "python train.py", string(items[1].data))
}

func TestBuildArtifacts_ParametersButNoRequirements(t *testing.T) {
	cfg := &runConfig{
		ExperimentName: "test",
		Compute:        &computeConfig{AcceleratorType: "GPU_1xH100", NumAccelerators: 1},
		Command:        new("echo hi"),
		Environment: &environmentConfig{
			Dependencies: dependencies{set: true, list: []string{"torch", "numpy"}},
			Version:      stringOrInt{set: true, raw: "5"},
		},
		Parameters: map[string]any{"lr": 0.1},
	}

	// Inline deps are not uploaded, so the artifacts are config, command, and params.
	items, err := buildArtifacts(cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{trainingConfigName, commandScriptName, hyperparametersName}, itemNames(items))
	assert.YAMLEq(t, `
experiment_name: test
compute:
  num_accelerators: 1
  accelerator_type: GPU_1xH100
environment:
  dependencies:
    - torch
    - numpy
  version: 5
command: echo hi
parameters:
  lr: 0.1
`, string(itemData(t, items, trainingConfigName)))
}

func TestBuildArtifacts_EnvVarsAndSecrets(t *testing.T) {
	cfg := &runConfig{
		Command:      new("echo hi"),
		EnvVariables: map[string]string{"WANDB": "demo"},
		Secrets:      map[string]string{"HF_TOKEN": "myscope/hf"},
	}

	items, err := buildArtifacts(cfg)
	require.NoError(t, err)
	assert.Subset(t, itemNames(items), []string{envVarsName, secretEnvVarsName})

	byName := map[string][]byte{}
	for _, it := range items {
		byName[it.name] = it.data
	}
	assert.JSONEq(t, `[{"name":"WANDB","value":"demo"}]`, string(byName[envVarsName]))
	assert.JSONEq(t, `[{"name":"HF_TOKEN","secret_scope":"myscope","secret_key":"hf"}]`, string(byName[secretEnvVarsName]))
}

func TestBuildArtifacts_FinalNormalizedConfigWithNestedOverrides(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", `
experiment_name: artifact-test
command: python train.py
compute:
  accelerator_type: GPU_1xH100
  num_accelerators: 1
parameters:
  model:
    hidden_size: 1024
  optimizer:
    name: adamw
mlflow_artifact_location: /Volumes/main/default/artifacts
`)
	cfg, err := loadRunConfigWithOverrides(t.Context(), path, []string{
		"compute.num_accelerators=4",
		"parameters.model.hidden_size=2048",
		"parameters.optimizer.learning_rate=0.001",
	})
	require.NoError(t, err)

	items, err := buildArtifacts(cfg)
	require.NoError(t, err)

	var uploaded map[string]any
	require.NoError(t, yaml.Unmarshal(itemData(t, items, trainingConfigName), &uploaded))
	assert.Equal(t, 4, uploaded["compute"].(map[string]any)["num_accelerators"])
	parameters := uploaded["parameters"].(map[string]any)
	assert.Equal(t, 2048, parameters["model"].(map[string]any)["hidden_size"])
	assert.InDelta(t, 0.001, parameters["optimizer"].(map[string]any)["learning_rate"], 0)
	assert.Equal(t, "adamw", parameters["optimizer"].(map[string]any)["name"])
	assert.Equal(t, "dbfs:/Volumes/main/default/artifacts", uploaded["mlflow_artifact_location"])
}

func TestBuildArtifacts_OversizeConfigRejected(t *testing.T) {
	_, err := buildArtifacts(&runConfig{
		Command:    new("x"),
		Parameters: map[string]any{"value": strings.Repeat("a", maxConfigYAMLBytes+1)},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "over the 1 MB limit")
}

func TestUploadArtifacts(t *testing.T) {
	w := &fakeWriter{}
	items := []uploadItem{{trainingConfigName, []byte("cfg")}, {commandScriptName, []byte("cmd")}}
	require.NoError(t, uploadArtifacts(t.Context(), w, items))
	assert.Equal(t, "cfg", w.written[trainingConfigName])
	assert.Equal(t, "cmd", w.written[commandScriptName])
}

// errWriter fails every Write, exercising the upload error path.
type errWriter struct{}

func (errWriter) Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error {
	return errors.New("boom")
}

func TestUploadArtifacts_WriteError(t *testing.T) {
	err := uploadArtifacts(t.Context(), errWriter{}, []uploadItem{{trainingConfigName, []byte("x")}})
	require.ErrorContains(t, err, "failed to upload "+trainingConfigName)
}
