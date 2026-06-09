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
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cacheDirEnv relocates the file cache, and with it the refresh lock, so tests
// never touch the user's real cache.
const cacheDirEnv = "DATABRICKS_CACHE_DIR"

func startReleaseServer(t *testing.T, tag, htmlURL string) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"` + htmlURL + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func interactiveContext(t *testing.T) context.Context {
	ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader(""), io.Discard, io.Discard))
	// gatherConditions reads dbr.RunsOnRuntime, which requires a detected context.
	return dbr.MockRuntime(ctx, dbr.Environment{})
}

func regularCmd(t *testing.T) *cobra.Command {
	c := &cobra.Command{Use: "deploy"}
	c.Flags().String("output", "text", "")
	return c
}

func TestShouldNotify(t *testing.T) {
	tests := []struct {
		name string
		c    notifyConditions
		want bool
	}{
		{"all clear", notifyConditions{}, true},
		{"development build", notifyConditions{developmentBuild: true}, false},
		{"cache disabled", notifyConditions{cacheDisabled: true}, false},
		{"opted out", notifyConditions{optedOut: true}, false},
		{"on databricks runtime", notifyConditions{onRuntime: true}, false},
		{"ci", notifyConditions{ci: true}, false},
		{"non-interactive", notifyConditions{nonInteractive: true}, false},
		{"json output", notifyConditions{jsonOutput: true}, false},
		{"exempt command", notifyConditions{exemptCommand: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldNotify(tt.c))
		})
	}
}

func TestGatherConditions(t *testing.T) {
	original := build.GetInfo().Version
	t.Cleanup(func() { build.SetBuildVersion(original) })
	build.SetBuildVersion("0.240.0")

	t.Run("all clear for an interactive release build", func(t *testing.T) {
		t.Setenv(ciEnv, "false")
		t.Setenv(disableEnv, "false")
		assert.True(t, shouldNotify(gatherConditions(interactiveContext(t), regularCmd(t))))
	})

	t.Run("CI env", func(t *testing.T) {
		t.Setenv(ciEnv, "true")
		t.Setenv(disableEnv, "false")
		assert.True(t, gatherConditions(interactiveContext(t), regularCmd(t)).ci)
	})

	t.Run("opt-out env", func(t *testing.T) {
		t.Setenv(ciEnv, "false")
		t.Setenv(disableEnv, "true")
		assert.True(t, gatherConditions(interactiveContext(t), regularCmd(t)).optedOut)
	})

	t.Run("cache disabled", func(t *testing.T) {
		t.Setenv(ciEnv, "false")
		t.Setenv(disableEnv, "false")
		t.Setenv(cacheEnabledEnv, "false")
		assert.True(t, gatherConditions(interactiveContext(t), regularCmd(t)).cacheDisabled)
	})

	t.Run("json output", func(t *testing.T) {
		t.Setenv(ciEnv, "false")
		t.Setenv(disableEnv, "false")
		c := regularCmd(t)
		require.NoError(t, c.Flags().Set("output", "json"))
		assert.True(t, gatherConditions(interactiveContext(t), c).jsonOutput)
	})

	t.Run("non-interactive", func(t *testing.T) {
		t.Setenv(ciEnv, "false")
		t.Setenv(disableEnv, "false")
		ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
		ctx = dbr.MockRuntime(ctx, dbr.Environment{})
		assert.True(t, gatherConditions(ctx, regularCmd(t)).nonInteractive)
	})

	t.Run("bare context without cmdio or dbr detection", func(t *testing.T) {
		// Help invocations skip PersistentPreRunE (no cmdio) and tests execute
		// commands without root.Execute (no dbr detection); neither may panic.
		t.Setenv(ciEnv, "false")
		t.Setenv(disableEnv, "false")
		c := gatherConditions(t.Context(), regularCmd(t))
		assert.True(t, c.nonInteractive)
		assert.False(t, c.onRuntime)
	})
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

func TestNoticeMessage(t *testing.T) {
	original := build.GetInfo().Version
	t.Cleanup(func() { build.SetBuildVersion(original) })

	warm := func(t *testing.T, current, latestTag, latestURL string) {
		t.Setenv(cacheDirEnv, t.TempDir())
		t.Setenv(gitHubAPIURLEnv, startReleaseServer(t, latestTag, latestURL))
		build.SetBuildVersion(current)
		require.NoError(t, refreshLatest(t.Context()))
	}

	t.Run("update available, nudged at most once per day", func(t *testing.T) {
		warm(t, "0.240.0", "v0.245.0", "https://example.test/releases/tag/v0.245.0")

		msg := noticeMessage(t.Context())
		assert.Contains(t, msg, "0.240.0 → 0.245.0")
		assert.Contains(t, msg, "https://example.test/releases/tag/v0.245.0")

		// Throttled the second time the same day.
		assert.Empty(t, noticeMessage(t.Context()))
	})

	t.Run("no notice when up to date", func(t *testing.T) {
		warm(t, "0.245.0", "v0.245.0", "https://example.test/x")
		assert.Empty(t, noticeMessage(t.Context()))
	})

	t.Run("no notice when cache is cold", func(t *testing.T) {
		t.Setenv(cacheDirEnv, t.TempDir())
		build.SetBuildVersion("0.240.0")
		assert.Empty(t, noticeMessage(t.Context()))
	})
}

func TestNoticeSuppressedOnError(t *testing.T) {
	assert.Empty(t, Notice(t.Context(), &cobra.Command{Use: "deploy"}, errors.New("boom")))
}

func TestRefreshLatestCachesFailure(t *testing.T) {
	original := build.GetInfo().Version
	t.Cleanup(func() { build.SetBuildVersion(original) })
	build.SetBuildVersion("0.240.0")
	t.Setenv(cacheDirEnv, t.TempDir())

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv(gitHubAPIURLEnv, srv.URL)

	require.Error(t, refreshLatest(t.Context()))
	require.Equal(t, int32(1), hits.Load())

	// The failure is cached: the next command neither retries the fetch nor
	// shows a notice until the entry expires.
	require.NoError(t, refreshLatest(t.Context()))
	assert.Equal(t, int32(1), hits.Load())
	assert.Empty(t, noticeMessage(t.Context()))
}

func TestTryAcquireRefreshLock(t *testing.T) {
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
