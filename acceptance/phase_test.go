package acceptance_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectTestPhases(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	files := map[string]string{
		"test.toml":        "Phase = 9\n",
		"alpha/test.toml":  "Local = true\n",
		"beta/test.toml":   "Phase = 1\n",
		"gamma/test.toml":  "Phase = 2\n",
		"zeta/test.toml":   "Local = true\nPhase = 1\n",
		"nested/test.toml": "Phase = 4\n",
	}

	for path, contents := range files {
		absPath := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
		require.NoError(t, os.WriteFile(absPath, []byte(contents), 0o644))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(root, "nested", "leaf"), 0o755))

	phases := collectTestPhases(t, []string{"gamma", "beta", "alpha", "zeta", "nested/leaf"})

	assert.Equal(t, []testPhase{
		{
			Phase: 0,
			Dirs:  []string{"alpha", "nested/leaf"},
		},
		{
			Phase: 1,
			Dirs:  []string{"beta", "zeta"},
		},
		{
			Phase: 2,
			Dirs:  []string{"gamma"},
		},
	}, phases)
	assert.Equal(t, []string{"alpha", "nested/leaf", "beta", "zeta", "gamma"}, flattenTestPhases(phases))
}

func TestPhaseSchedulerWaitsForPreviousPhase(t *testing.T) {
	scheduler := newPhaseScheduler([]testPhase{
		{
			Phase: 0,
			Dirs:  []string{"a", "b"},
		},
		{
			Phase: 1,
			Dirs:  []string{"c"},
		},
	})

	released := make(chan struct{})
	go func() {
		scheduler.Wait(1)
		close(released)
	}()

	select {
	case <-released:
		t.Fatal("phase 1 should stay blocked until phase 0 completes")
	case <-time.After(20 * time.Millisecond):
	}

	scheduler.Done(0)

	select {
	case <-released:
		t.Fatal("phase 1 should stay blocked until all phase 0 tests complete")
	case <-time.After(20 * time.Millisecond):
	}

	scheduler.Done(0)

	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("phase 1 was not released")
	}
}
