package aircmd

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWriter records artifact writes in place of a workspace filer.
type fakeWriter struct {
	mu         sync.Mutex
	written    map[string]string
	modes      map[string][]filer.WriteMode
	mkdirPaths []string
}

func (f *fakeWriter) Mkdir(ctx context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mkdirPaths = append(f.mkdirPaths, name)
	return nil
}

func (f *fakeWriter) Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.mkdirPaths) == 0 {
		return errors.New("write called before mkdir")
	}
	if f.written == nil {
		f.written = map[string]string{}
		f.modes = map[string][]filer.WriteMode{}
	}
	f.written[name] = string(data)
	f.modes[name] = append([]filer.WriteMode(nil), mode...)
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
	assert.Equal(t, []string{"."}, w.mkdirPaths)
	assert.Equal(t, "cfg", w.written[trainingConfigName])
	assert.Equal(t, "cmd", w.written[commandScriptName])
	assert.Equal(t, []filer.WriteMode{filer.OverwriteIfExists}, w.modes[trainingConfigName])
	assert.Equal(t, []filer.WriteMode{filer.OverwriteIfExists}, w.modes[commandScriptName])
}

// errWriter fails every Write, exercising the upload error path.
type errWriter struct{}

func (errWriter) Mkdir(ctx context.Context, name string) error { return nil }

func (errWriter) Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error {
	return errors.New("boom")
}

func TestUploadArtifacts_WriteError(t *testing.T) {
	err := uploadArtifacts(t.Context(), errWriter{}, []uploadItem{{trainingConfigName, []byte("x")}})
	require.ErrorContains(t, err, "failed to upload "+trainingConfigName)
}

type mkdirErrWriter struct{}

func (mkdirErrWriter) Mkdir(ctx context.Context, name string) error {
	return errors.New("boom")
}

func (mkdirErrWriter) Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error {
	return errors.New("write should not be called")
}

func TestUploadArtifacts_MkdirError(t *testing.T) {
	err := uploadArtifacts(t.Context(), mkdirErrWriter{}, []uploadItem{{trainingConfigName, []byte("x")}})
	require.ErrorContains(t, err, "failed to create launch directory")
}

// overlappingWriter blocks each write until two writes are in flight. A
// sequential uploader cannot satisfy the barrier.
type overlappingWriter struct {
	mu          sync.Mutex
	active      int
	bothStarted chan struct{}
	release     chan struct{}
}

func (w *overlappingWriter) Mkdir(ctx context.Context, name string) error { return nil }

func (w *overlappingWriter) Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error {
	w.mu.Lock()
	w.active++
	if w.active == 2 {
		close(w.bothStarted)
	}
	w.mu.Unlock()

	select {
	case <-w.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestUploadArtifacts_WritesConcurrently(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	w := &overlappingWriter{bothStarted: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		done <- uploadArtifacts(ctx, w, []uploadItem{
			{trainingConfigName, []byte("cfg")},
			{commandScriptName, []byte("cmd")},
		})
	}()

	select {
	case <-w.bothStarted:
		close(w.release)
	case <-ctx.Done():
		close(w.release)
		t.Fatal("artifact writes did not overlap")
	}
	require.NoError(t, <-done)
}
