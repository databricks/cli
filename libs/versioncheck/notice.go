package versioncheck

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/cmdio"
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
)

// StartBackgroundRefresh fetches the latest released version in the background
// (at most once per day) and stores it in the cache, so a later command can
// show the notice without a blocking network call. It is a no-op when the
// notice would be suppressed anyway. Safe to call on every invocation.
func StartBackgroundRefresh(ctx context.Context, cmd *cobra.Command) {
	if suppressed(ctx, cmd) {
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

// refreshLatest fetches and caches the latest released version, but only when
// the cached value is older than cacheTTL (or absent).
func refreshLatest(ctx context.Context) error {
	c := cache.NewCache(ctx, cacheComponent, cacheTTL, nil)
	_, err := cache.GetOrCompute(ctx, c, latestFingerprint, fetchLatestVersion)
	return err
}

// Notice returns a one-line "new version available" message to print after a
// command, or an empty string when nothing should be shown. It reads the
// cached latest version (never the network) and nudges at most once per day.
// runErr is the command's result; the notice is suppressed when it is non-nil.
func Notice(ctx context.Context, cmd *cobra.Command, runErr error) (msg string) {
	// A failure here must never affect the user's command, which has already
	// completed by the time this runs.
	defer func() {
		if r := recover(); r != nil {
			log.Debugf(ctx, "version check: notice panic: %v", r)
			msg = ""
		}
	}()

	if runErr != nil || suppressed(ctx, cmd) {
		return ""
	}

	c := cache.NewCache(ctx, cacheComponent, cacheTTL, nil)
	latest, ok := cache.Get[string](ctx, c, latestFingerprint)
	if !ok {
		// Not refreshed yet (cold or stale cache); a background refresh will
		// warm it for a later command.
		return ""
	}

	current := build.GetInfo().Version
	if !isNewer(current, latest) {
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

func noticeText(ctx context.Context, current, latest, upgradeCommand string) string {
	msg := fmt.Sprintf("A new version of the Databricks CLI is available: %s (you have %s).", latest, current)
	if upgradeCommand != "" {
		msg += fmt.Sprintf(" Run `%s` to upgrade.", upgradeCommand)
	} else {
		msg += " See https://github.com/databricks/cli/releases to upgrade."
	}
	return cmdio.Yellow(ctx, msg)
}

// suppressed reports whether the passive notice should be withheld. It errs
// toward staying quiet: anything non-interactive, scripted, or opted out is
// suppressed.
func suppressed(ctx context.Context, cmd *cobra.Command) bool {
	if isDevelopmentBuild(build.GetInfo()) {
		return true
	}
	if disabled, ok := env.GetBool(ctx, disableEnv); ok && disabled {
		return true
	}
	// Honor the common CI convention even when a pseudo-TTY is allocated.
	// https://github.blog/changelog/2020-04-15-github-actions-sets-the-ci-environment-variable-to-true/
	if ci, ok := env.GetBool(ctx, "CI"); ok && ci {
		return true
	}
	// No usable terminal (pipes, cron, stderr redirected).
	if cmdio.GetInteractiveMode(ctx) == cmdio.InteractiveModeNone {
		return true
	}
	if isJSONOutput(cmd) {
		return true
	}
	return isExemptCommand(cmd)
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
