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

func TestBuildArtifacts_InlineRequirementsAndParameters(t *testing.T) {
	path := writeConfigFile(t, "run.yaml", "x: y\n")
	cfg := &runConfig{
		Command: new("echo hi"),
		Environment: &environmentConfig{
			Dependencies: dependencies{set: true, isList: true, list: []string{"torch", "numpy"}},
			Version:      stringOrInt{set: true, raw: "5"},
		},
		Parameters: map[string]any{"lr": 0.1},
	}

	items, err := buildArtifacts(cfg, path)
	require.NoError(t, err)
	assert.Equal(t, []string{trainingConfigName, commandScriptName, requirementsName, hyperparametersName}, itemNames(items))

	var reqIdx int
	for i, it := range items {
		if it.name == requirementsName {
			reqIdx = i
		}
	}
	req := string(items[reqIdx].data)
	assert.Contains(t, req, "version: \"5\"")
	assert.Contains(t, req, "- torch")
}

func TestBuildArtifacts_RequirementsFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "run.yaml"), []byte("x: y\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reqs.yaml"), []byte("version: 4\n"), 0o600))
	cfg := &runConfig{
		Command:     new("echo hi"),
		Environment: &environmentConfig{Dependencies: dependencies{set: true, isList: false, path: "reqs.yaml"}},
	}

	items, err := buildArtifacts(cfg, filepath.Join(dir, "run.yaml"))
	require.NoError(t, err)
	assert.Contains(t, itemNames(items), requirementsName)
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

func TestBuildArtifacts_MissingRequirementsFile(t *testing.T) {
	cfgPath := writeConfigFile(t, "run.yaml", "x: y\n")
	cfg := &runConfig{
		Command:     new("echo hi"),
		Environment: &environmentConfig{Dependencies: dependencies{set: true, isList: false, path: "nope.yaml"}},
	}
	_, err := buildArtifacts(cfg, cfgPath)
	require.ErrorContains(t, err, "failed to read requirements file")
}
