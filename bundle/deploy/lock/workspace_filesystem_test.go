package lock_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memFiler is a simple in-memory filer for testing.
type memFiler struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newMemFiler() *memFiler {
	return &memFiler{files: make(map[string][]byte)}
}

func (m *memFiler) Read(_ context.Context, path string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memFiler) Write(_ context.Context, path string, r io.Reader, _ ...filer.WriteMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.files[path] = data
	return nil
}

func (m *memFiler) Delete(_ context.Context, _ string, _ ...filer.DeleteMode) error {
	return nil
}

func (m *memFiler) ReadDir(_ context.Context, _ string) ([]fs.DirEntry, error) {
	return nil, nil
}

func (m *memFiler) Mkdir(_ context.Context, _ string) error {
	return nil
}

func (m *memFiler) Stat(_ context.Context, _ string) (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func TestIncrementDeploymentVersion_FirstDeploy(t *testing.T) {
	ctx := t.Context()
	f := newMemFiler()
	version, err := lock.IncrementDeploymentVersion(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)
}

func TestIncrementDeploymentVersion_IncrementsOnEachCall(t *testing.T) {
	ctx := t.Context()
	f := newMemFiler()

	for i := range int64(5) {
		version, err := lock.IncrementDeploymentVersion(ctx, f)
		require.NoError(t, err)
		assert.Equal(t, i+1, version)
	}
}
