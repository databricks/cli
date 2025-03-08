package dbr

import (
	"context"
	"io/fs"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/fakefs"
	"github.com/stretchr/testify/assert"
)

func requireLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("skipping test on %s", runtime.GOOS)
	}
}

func configureStatFunc(t *testing.T, fi fs.FileInfo, err error) {
	originalFunc := statFunc
	statFunc = func(name string) (fs.FileInfo, error) {
		assert.Equal(t, "/databricks", name)
		return fi, err
	}

	t.Cleanup(func() {
		statFunc = originalFunc
	})
}

func TestDetect_NotLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping test on Linux OS")
	}

	ctx := context.Background()
	assert.Equal(t, Environment{}, detect(ctx))
}

func TestDetect_Env(t *testing.T) {
	requireLinux(t)

	// Configure other checks to pass.
	configureStatFunc(t, fakefs.FileInfo{FakeDir: true}, nil)

	t.Run("empty", func(t *testing.T) {
		ctx := env.Set(context.Background(), "DATABRICKS_RUNTIME_VERSION", "")
		assert.Equal(t, Environment{}, detect(ctx))
	})

	t.Run("non-empty cluster", func(t *testing.T) {
		ctx := env.Set(context.Background(), "DATABRICKS_RUNTIME_VERSION", "15.4")
		assert.Equal(t, Environment{
			IsDbr:   true,
			Version: "15.4",
		}, detect(ctx))
	})

	t.Run("non-empty serverless", func(t *testing.T) {
		ctx := env.Set(context.Background(), "DATABRICKS_RUNTIME_VERSION", "client.1.13")
		assert.Equal(t, Environment{
			IsDbr:   true,
			Version: "client.1.13",
		}, detect(ctx))
	})
}

func TestDetect_Stat(t *testing.T) {
	requireLinux(t)

	// Configure other checks to pass.
	ctx := env.Set(context.Background(), "DATABRICKS_RUNTIME_VERSION", "non-empty")

	t.Run("error", func(t *testing.T) {
		configureStatFunc(t, nil, fs.ErrNotExist)
		assert.Equal(t, Environment{}, detect(ctx))
	})

	t.Run("not a directory", func(t *testing.T) {
		configureStatFunc(t, fakefs.FileInfo{}, nil)
		assert.Equal(t, Environment{}, detect(ctx))
	})

	t.Run("directory", func(t *testing.T) {
		configureStatFunc(t, fakefs.FileInfo{FakeDir: true}, nil)
		assert.Equal(t, Environment{
			IsDbr:   true,
			Version: "non-empty",
		}, detect(ctx))
	})
}
