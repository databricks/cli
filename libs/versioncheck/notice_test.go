package versioncheck

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startReleaseServer(t *testing.T, tag string) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func interactiveContext(t *testing.T) context.Context {
	return cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader(""), io.Discard, io.Discard))
}

func TestIsExemptCommand(t *testing.T) {
	withParent := func(parent, child string) *cobra.Command {
		p := &cobra.Command{Use: parent}
		c := &cobra.Command{Use: child}
		p.AddCommand(c)
		return c
	}

	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{"version", &cobra.Command{Use: "version"}, true},
		{"version check subcommand", withParent("version", "check"), true},
		{"completion", &cobra.Command{Use: "completion"}, true},
		{"help", &cobra.Command{Use: "help"}, true},
		{"shell completion request", &cobra.Command{Use: cobra.ShellCompRequestCmd}, true},
		{"regular command", &cobra.Command{Use: "bundle"}, false},
		{"regular subcommand", withParent("bundle", "deploy"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isExemptCommand(tt.cmd))
		})
	}
}

func TestSuppressed(t *testing.T) {
	original := build.GetInfo().Version
	t.Cleanup(func() { build.SetBuildVersion(original) })

	regularCmd := func(output string) *cobra.Command {
		c := &cobra.Command{Use: "deploy"}
		c.Flags().String("output", "text", "")
		if output != "" {
			require.NoError(t, c.Flags().Set("output", output))
		}
		return c
	}

	tests := []struct {
		name         string
		buildVersion string
		ci           string
		disable      string
		cacheEnabled string
		interactive  bool
		cmd          *cobra.Command
		want         bool
	}{
		{"interactive release build", "0.240.0", "false", "false", "", true, regularCmd(""), false},
		{"development build", "0.0.0-dev+abc", "false", "false", "", true, regularCmd(""), true},
		{"opt-out env", "0.240.0", "false", "true", "", true, regularCmd(""), true},
		{"CI env", "0.240.0", "true", "false", "", true, regularCmd(""), true},
		{"cache disabled", "0.240.0", "false", "false", "false", true, regularCmd(""), true},
		{"non-interactive", "0.240.0", "false", "false", "", false, regularCmd(""), true},
		{"json output", "0.240.0", "false", "false", "", true, regularCmd("json"), true},
		{"exempt command", "0.240.0", "false", "false", "", true, &cobra.Command{Use: "version"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build.SetBuildVersion(tt.buildVersion)
			t.Setenv("CI", tt.ci)
			t.Setenv(disableEnv, tt.disable)
			if tt.cacheEnabled != "" {
				t.Setenv(cacheEnabledEnv, tt.cacheEnabled)
			}

			ctx := interactiveContext(t)
			if !tt.interactive {
				ctx, _ = cmdio.NewTestContextWithStdout(t.Context())
			}

			assert.Equal(t, tt.want, suppressed(ctx, tt.cmd))
		})
	}
}

func TestTryAcquireRefreshLock(t *testing.T) {
	// Isolate the lock to this test's temp dir.
	dir := t.TempDir()
	t.Setenv(cacheDirEnv, dir)
	ctx := t.Context()

	release, ok := tryAcquireRefreshLock(ctx)
	require.True(t, ok)

	// A second caller is locked out while the first holds the lock.
	_, ok2 := tryAcquireRefreshLock(ctx)
	assert.False(t, ok2)

	// After release, the lock can be acquired again.
	release()
	release2, ok3 := tryAcquireRefreshLock(ctx)
	require.True(t, ok3)
	release2()

	// A lock left behind by a crashed holder (old mtime, never released) is
	// reclaimed once it is older than lockStaleAfter.
	lockPath := filepath.Join(dir, refreshLockName)
	require.NoError(t, os.WriteFile(lockPath, nil, 0o600))
	old := time.Now().Add(-2 * lockStaleAfter)
	require.NoError(t, os.Chtimes(lockPath, old, old))
	release3, ok4 := tryAcquireRefreshLock(ctx)
	require.True(t, ok4)
	release3()
}

func TestNotice(t *testing.T) {
	original := build.GetInfo().Version
	t.Cleanup(func() { build.SetBuildVersion(original) })

	// neutralEnv isolates a subtest from the ambient environment (CI runners
	// set CI=true) and points the cache + release lookup at test-local targets.
	neutralEnv := func(t *testing.T, latestTag string) {
		t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
		t.Setenv("CI", "false")
		t.Setenv(disableEnv, "false")
		t.Setenv(gitHubAPIURLEnv, startReleaseServer(t, latestTag))
	}

	regularCmd := func() *cobra.Command {
		c := &cobra.Command{Use: "deploy"}
		c.Flags().String("output", "text", "")
		return c
	}

	t.Run("update available, nudged at most once per day", func(t *testing.T) {
		neutralEnv(t, "v0.245.0")
		build.SetBuildVersion("0.240.0")
		require.NoError(t, refreshLatest(t.Context()))

		msg := Notice(interactiveContext(t), regularCmd(), nil)
		assert.Contains(t, msg, "0.245.0")
		assert.Contains(t, msg, "0.240.0")

		// A second invocation the same day is throttled.
		assert.Empty(t, Notice(interactiveContext(t), regularCmd(), nil))
	})

	t.Run("no notice when up to date", func(t *testing.T) {
		neutralEnv(t, "v0.240.0")
		build.SetBuildVersion("0.240.0")
		require.NoError(t, refreshLatest(t.Context()))

		assert.Empty(t, Notice(interactiveContext(t), regularCmd(), nil))
	})

	t.Run("no notice when cache is cold", func(t *testing.T) {
		neutralEnv(t, "v0.245.0")
		build.SetBuildVersion("0.240.0")
		// No refreshLatest call: nothing cached yet.

		assert.Empty(t, Notice(interactiveContext(t), regularCmd(), nil))
	})

	t.Run("no notice on command error", func(t *testing.T) {
		neutralEnv(t, "v0.245.0")
		build.SetBuildVersion("0.240.0")
		require.NoError(t, refreshLatest(t.Context()))

		assert.Empty(t, Notice(interactiveContext(t), regularCmd(), errors.New("boom")))
	})

	t.Run("no notice when suppressed", func(t *testing.T) {
		neutralEnv(t, "v0.245.0")
		t.Setenv("CI", "true")
		build.SetBuildVersion("0.240.0")
		require.NoError(t, refreshLatest(t.Context()))

		assert.Empty(t, Notice(interactiveContext(t), regularCmd(), nil))
	})
}
