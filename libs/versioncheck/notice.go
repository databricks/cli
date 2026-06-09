package versioncheck

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

const (
	// cacheComponent names the on-disk cache bucket (under the bundle file
	// cache). cacheTTL drives both the once-per-day network check and the
	// once-per-day display: the file cache treats an entry older than the TTL
	// as a miss, so the latest version is refetched and the user is nudged at
	// most once per day. The cache is namespaced by CLI version, so it resets
	// automatically after an upgrade.
	cacheComponent = "update-check"
	cacheTTL       = 24 * time.Hour

	// backgroundTimeout bounds the background fetch so an abandoned goroutine
	// doesn't linger when GitHub is slow or unreachable.
	backgroundTimeout = 3 * time.Second

	latestFingerprint   = "latest-release"
	notifiedFingerprint = "notified"

	// disableEnv is an explicit opt-out for the passive notice.
	disableEnv = "DATABRICKS_CLI_DISABLE_UPDATE_CHECK"

	// cacheEnabledEnv mirrors libs/cache's env knob. We read it directly to
	// skip the check entirely when caching is turned off: with the cache in
	// measurement-only mode, every command would otherwise refetch.
	cacheEnabledEnv = "DATABRICKS_CACHE_ENABLED"

	// ciEnv: virtually all CI providers set this, so suppress the notice there
	// even when a pseudo-TTY is allocated.
	// https://docs.github.com/actions/learn-github-actions/variables
	ciEnv = "CI"

	// refreshLockName is the single-flight sentinel: across many concurrent CLI
	// processes only the one that creates it fetches, so a fleet starting at
	// once sends at most one request to GitHub per machine.
	refreshLockName = "databricks-cli-update-check.lock"
	// lockStaleAfter reclaims a lock left behind when the process exited while
	// the background goroutine was still fetching, so its deferred release never
	// ran. It must comfortably exceed backgroundTimeout so a live fetch is never
	// mistaken for an abandoned lock.
	lockStaleAfter = 30 * time.Second
)

// StartBackgroundRefresh fetches the latest release in the background (at most
// once per day) and stores it in the cache, so a later command can show the
// notice without a blocking network call. It is a no-op when the notice would
// be suppressed anyway. Safe to call on every invocation.
func StartBackgroundRefresh(ctx context.Context, cmd *cobra.Command) {
	if !notifyEnabled(ctx, cmd) {
		return
	}
	go func() {
		// A panic in this best-effort goroutine must never crash the CLI.
		defer func() {
			if r := recover(); r != nil {
				log.Debugf(ctx, "version check: background refresh panic: %v", r)
			}
		}()
		ctx, cancel := context.WithTimeout(ctx, backgroundTimeout)
		defer cancel()
		if err := refreshLatest(ctx); err != nil {
			log.Debugf(ctx, "version check: background refresh failed: %v", err)
		}
	}()
}

// refreshLatest fetches and caches the latest release, but only when the cached
// value is older than cacheTTL (or absent), and only one process at a time per
// machine performs the fetch.
func refreshLatest(ctx context.Context) error {
	c := cache.NewCache(ctx, cacheComponent, cacheTTL, nil)

	// Common path: a fresh entry means no fetch and no lock churn.
	if _, fresh := cache.Get[latestRelease](ctx, c, latestFingerprint); fresh {
		return nil
	}

	// Single-flight across processes so a fleet of CLIs doesn't stampede GitHub.
	release, ok := tryAcquireRefreshLock(ctx)
	if !ok {
		return nil
	}
	defer release()

	// GetOrCompute re-checks freshness under the lock, so a process that lost
	// the race reads the just-written value instead of fetching again.
	_, err := cache.GetOrCompute(ctx, c, latestFingerprint, fetchLatestRelease)
	if err != nil {
		// Cache the failure too: an offline or airgapped machine should not
		// retry GitHub on every command. The empty entry reads as "no update
		// available" and expires with the normal TTL.
		cache.Put(ctx, c, latestFingerprint, latestRelease{})
	}
	return err
}

// tryAcquireRefreshLock claims a best-effort, machine-wide single-flight lock
// via an O_EXCL sentinel file. It returns (release, true) when the caller may
// fetch, or (nil, false) when another process holds a fresh lock or the lock
// can't be created. A lock older than lockStaleAfter is treated as abandoned
// (the holder crashed) and stolen. This is best-effort coordination to prevent
// a thundering herd, not a correctness-critical lock; a dedicated flock library
// would avoid the staleness heuristic but isn't a dependency here.
func tryAcquireRefreshLock(ctx context.Context) (func(), bool) {
	// The lock lives in the cache root: per-user, so another user's stale lock
	// on a shared machine can never block (or be stolen by) this one. The
	// directory exists by the time this runs because refreshLatest constructs
	// the cache first; if that failed, failing to lock here is the right call.
	dir, err := cache.BaseDir(ctx)
	if err != nil {
		return nil, false
	}
	lockPath := filepath.Join(dir, refreshLockName)

	for range 2 {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, true
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, false
		}
		info, statErr := os.Stat(lockPath)
		if statErr != nil || time.Since(info.ModTime()) <= lockStaleAfter {
			return nil, false
		}
		_ = os.Remove(lockPath) // stale; steal it and retry once
	}
	return nil, false
}

// Notice returns a "new release available" message to print after a command, or
// an empty string when nothing should be shown. runErr is the command's result;
// the notice is suppressed when it is non-nil so a hint never stacks on an error.
func Notice(ctx context.Context, cmd *cobra.Command, runErr error) (msg string) {
	// A failure here must never affect the user's command, which has already
	// completed by the time this runs.
	defer func() {
		if r := recover(); r != nil {
			log.Debugf(ctx, "version check: notice panic: %v", r)
			msg = ""
		}
	}()

	if runErr != nil || !notifyEnabled(ctx, cmd) {
		return ""
	}
	return noticeMessage(ctx)
}

// noticeMessage renders the advisory from the cached latest release, or returns
// "" when the CLI is current, the cache is cold, or the once-per-day nudge has
// already fired. It reads the cache only, never the network.
func noticeMessage(ctx context.Context) string {
	c := cache.NewCache(ctx, cacheComponent, cacheTTL, nil)
	latest, ok := cache.Get[latestRelease](ctx, c, latestFingerprint)
	if !ok {
		// Not refreshed yet (cold or stale cache); a background refresh will
		// warm it for a later command.
		return ""
	}

	current := build.GetInfo().Version
	if !isNewer(current, latest.Version) {
		return ""
	}

	// Nudge at most once per cacheTTL: a fresh "notified" marker means we
	// already showed the notice today.
	if _, notified := cache.Get[bool](ctx, c, notifiedFingerprint); notified {
		return ""
	}
	cache.Put(ctx, c, notifiedFingerprint, true)

	_, command := DetectInstallMethod(ctx)
	return noticeText(ctx, current, latest, command)
}

// noticeText formats the advisory: the version delta, a link to the release for
// the changelog, and the upgrade command for the detected install method.
func noticeText(ctx context.Context, current string, latest latestRelease, upgradeCommand string) string {
	msg := fmt.Sprintf("A new release of the Databricks CLI is available: %s → %s", current, latest.Version)
	if latest.URL != "" {
		msg += "\n" + latest.URL
	}
	if upgradeCommand != "" {
		msg += "\nTo upgrade, run: " + upgradeCommand
	} else {
		msg += "\nSee https://github.com/databricks/cli/releases to upgrade."
	}
	return cmdio.Yellow(ctx, msg)
}

// notifyConditions captures the environment inputs for the passive-notice
// decision, keeping the policy (shouldNotify) a pure, table-testable function.
type notifyConditions struct {
	developmentBuild bool // dev/snapshot build: nothing to upgrade to
	cacheDisabled    bool // file cache off: nothing to store or read
	optedOut         bool // DATABRICKS_CLI_DISABLE_UPDATE_CHECK
	onRuntime        bool // Databricks Runtime: version is platform-managed
	ci               bool // continuous integration
	nonInteractive   bool // no usable terminal (pipes, cron, redirected)
	jsonOutput       bool // machine-readable output kept clean for scripts
	exemptCommand    bool // version/completion/help: notice would be noise
}

// shouldNotify reports whether the passive notice should run for the given
// conditions. It errs toward silence: any single reason suppresses it.
func shouldNotify(c notifyConditions) bool {
	switch {
	case c.developmentBuild, c.cacheDisabled, c.optedOut, c.onRuntime,
		c.ci, c.nonInteractive, c.jsonOutput, c.exemptCommand:
		return false
	}
	return true
}

// notifyEnabled gathers the current conditions and applies the policy.
func notifyEnabled(ctx context.Context, cmd *cobra.Command) bool {
	return shouldNotify(gatherConditions(ctx, cmd))
}

func gatherConditions(ctx context.Context, cmd *cobra.Command) notifyConditions {
	optedOut, _ := env.GetBool(ctx, disableEnv)
	ci, _ := env.GetBool(ctx, ciEnv)
	cacheDisabled := false
	if enabled, ok := env.GetBool(ctx, cacheEnabledEnv); ok && !enabled {
		cacheDisabled = true
	}
	// Both probes below panic on a context that skipped the usual command
	// setup, which legitimately happens here: cobra resolves --help and bare
	// invocations before PersistentPreRunE installs cmdio, and tests execute
	// commands without root.Execute's dbr detection. Missing IO means we have
	// no terminal to print to (suppress); unknown runtime means not DBR.
	nonInteractive := !cmdio.HasIO(ctx) || cmdio.GetInteractiveMode(ctx) == cmdio.InteractiveModeNone
	onRuntime := dbr.HasDetection(ctx) && dbr.RunsOnRuntime(ctx)
	return notifyConditions{
		developmentBuild: isDevelopmentBuild(build.GetInfo()),
		cacheDisabled:    cacheDisabled,
		optedOut:         optedOut,
		onRuntime:        onRuntime,
		ci:               ci,
		nonInteractive:   nonInteractive,
		jsonOutput:       isJSONOutput(cmd),
		exemptCommand:    isExemptCommand(cmd),
	}
}

func isJSONOutput(cmd *cobra.Command) bool {
	f := cmd.Flag("output")
	return f != nil && f.Value.String() == "json"
}

// isExemptCommand suppresses the notice for commands where it would be noise or
// would corrupt machine-readable output: the version commands themselves, shell
// completion, and help.
func isExemptCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "version", "completion", "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return true
		}
	}
	return false
}
