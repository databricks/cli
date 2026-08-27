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

func TestBuildArtifacts_CommandAndConfig(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", minimalConfig)
	cfg := &runConfig{Command: new("python train.py")}

	items, err := buildArtifacts(cfg, path)
	require.NoError(t, err)
	assert.Equal(t, []string{trainingConfigName, commandScriptName}, itemNames(items))
	assert.Equal(t, minimalConfig, string(items[0].data))
	assert.Equal(t, "python train.py", string(items[1].data))
}

func TestBuildArtifacts_ParametersButNoRequirements(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", "x: y\n")
	cfg := &runConfig{
		Command: new("echo hi"),
		Environment: &environmentConfig{
			Dependencies: dependencies{set: true, list: []string{"torch", "numpy"}},
			Version:      stringOrInt{set: true, raw: "5"},
		},
		Parameters: map[string]any{"lr": 0.1},
	}

	// Inline deps are not uploaded, so the artifacts are config, command, and params.
	items, err := buildArtifacts(cfg, path)
	require.NoError(t, err)
	assert.Equal(t, []string{trainingConfigName, commandScriptName, hyperparametersName}, itemNames(items))
}

func TestBuildArtifacts_FileRequirements(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "run.yaml")
	requirementsPath := filepath.Join(dir, "deps.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("x: y\n"), 0o600))
	require.NoError(t, os.WriteFile(requirementsPath, []byte("version: 5\ndependencies:\n  - torch\n"), 0o600))
	cfg := &runConfig{
		Command: new("echo hi"),
		Environment: &environmentConfig{Dependencies: dependencies{
			set: true, resolvedPath: requirementsPath,
		}},
	}

	items, err := buildArtifacts(cfg, configPath)
	require.NoError(t, err)
	assert.Equal(t, []string{trainingConfigName, commandScriptName, requirementsName}, itemNames(items))
	assert.Equal(t, "version: 5\ndependencies:\n  - torch\n", string(items[2].data))
}

func TestBuildArtifacts_EnvVarsAndSecrets(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", "x: y\n")
	cfg := &runConfig{
		Command:      new("echo hi"),
		EnvVariables: map[string]string{"WANDB": "demo"},
		Secrets:      map[string]string{"HF_TOKEN": "myscope/hf"},
	}

	items, err := buildArtifacts(cfg, path)
	require.NoError(t, err)
	assert.Subset(t, itemNames(items), []string{envVarsName, secretEnvVarsName})

	byName := map[string][]byte{}
	for _, it := range items {
		byName[it.name] = it.data
	}
	assert.JSONEq(t, `[{"name":"WANDB","value":"demo"}]`, string(byName[envVarsName]))
	assert.JSONEq(t, `[{"name":"HF_TOKEN","secret_scope":"myscope","secret_key":"hf"}]`, string(byName[secretEnvVarsName]))
}

func TestBuildArtifacts_OversizeConfigRejected(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", strings.Repeat("a", maxConfigYAMLBytes+1))
	_, err := buildArtifacts(&runConfig{Command: new("x")}, path)
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
