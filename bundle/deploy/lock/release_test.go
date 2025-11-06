package lock_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/libs/filer"
	lockpkg "github.com/databricks/cli/libs/locker"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestLocker creates a locker with a filer for testing
func createTestLocker(f filer.Filer, targetDir string) *lockpkg.Locker {
	l := &lockpkg.Locker{
		TargetDir: targetDir,
		Active:    false,
		State: &lockpkg.LockState{
			ID:   uuid.New(),
			User: "test-user",
		},
	}
	// Use reflection to set the private filer field
	v := reflect.ValueOf(l).Elem()
	filerField := v.FieldByName("filer")
	filerField = reflect.NewAt(filerField.Type(), filerField.Addr().UnsafePointer()).Elem()
	filerField.Set(reflect.ValueOf(f))
	return l
}

func TestReleaseIdempotent(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory for the filer
	tmpDir := t.TempDir()
	f, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)

	// Create a bundle with a locker
	enabled := true
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "test",
				Deployment: config.Deployment{
					Lock: config.Lock{
						Enabled: &enabled,
					},
				},
			},
		},
	}

	// Initialize locker with the filer
	locker := createTestLocker(f, tmpDir)
	b.Locker = locker

	// Acquire lock
	err = locker.Lock(ctx, false)
	require.NoError(t, err)
	assert.True(t, locker.Active)

	// Verify lock file exists
	_, err = f.Stat(ctx, "deploy.lock")
	require.NoError(t, err)

	// First release - should succeed
	mutator := lock.Release()
	diags := bundle.Apply(ctx, b, mutator)
	require.NoError(t, diags.Error())
	assert.False(t, locker.Active)

	// Verify lock file is deleted
	_, err = f.Stat(ctx, "deploy.lock")
	require.Error(t, err)

	// Second release - should be idempotent and succeed
	diags = bundle.Apply(ctx, b, mutator)
	require.NoError(t, diags.Error())
	assert.False(t, locker.Active)

	// Third release - should still be idempotent and succeed
	diags = bundle.Apply(ctx, b, mutator)
	require.NoError(t, diags.Error())
	assert.False(t, locker.Active)
}

func TestReleaseFileAlreadyDeleted(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory for the filer
	tmpDir := t.TempDir()
	f, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)

	// Create a bundle with a locker
	enabled := true
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "test",
				Deployment: config.Deployment{
					Lock: config.Lock{
						Enabled: &enabled,
					},
				},
			},
		},
	}

	// Initialize locker with the filer
	locker := createTestLocker(f, tmpDir)
	b.Locker = locker

	// Acquire lock
	err = locker.Lock(ctx, false)
	require.NoError(t, err)
	assert.True(t, locker.Active)

	// Verify lock file exists
	_, err = f.Stat(ctx, "deploy.lock")
	require.NoError(t, err)

	// Manually delete lock file
	err = f.Delete(ctx, "deploy.lock")
	require.NoError(t, err)

	// Release lock - should succeed even though lock file is already deleted
	mutator := lock.Release()
	diags := bundle.Apply(ctx, b, mutator)
	require.NoError(t, diags.Error())
	assert.False(t, locker.Active)
}

func TestReleaseWhenAnotherProcessHoldsLock(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory for the filer
	tmpDir := t.TempDir()
	f, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)

	// Create two lockers - simulating two different processes
	locker1 := createTestLocker(f, tmpDir)
	locker2 := createTestLocker(f, tmpDir)

	// First locker acquires the lock
	err = locker1.Lock(ctx, false)
	require.NoError(t, err)
	assert.True(t, locker1.Active)

	// Create bundle with second locker (different process)
	enabled := true
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "test",
				Deployment: config.Deployment{
					Lock: config.Lock{
						Enabled: &enabled,
					},
				},
			},
		},
		Locker: locker2,
	}

	// Set locker2 as active (simulating it thinks it has the lock, but it doesn't)
	locker2.Active = true

	// Try to release with locker2 - should error because locker1 holds the lock
	mutator := lock.Release()
	diags := bundle.Apply(ctx, b, mutator)
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "deploy lock acquired by")
}
