package root

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/upgradecheck"
	"github.com/spf13/cobra"
)

const versionCheckCacheName = "cli-version-check.json"

// startUpgradeCheck refreshes the cached latest-version record in the background
// when it is stale. It never blocks the command: the refresh runs in a goroutine
// and the upgrade notice (printed later by [printUpgradeNotice]) is rendered from
// whatever the cache holds when the command finishes.
func startUpgradeCheck(ctx context.Context, cmd *cobra.Command) {
	if !upgradeCheckEnabled(ctx, cmd) {
		return
	}
	cacheFile, err := versionCheckCacheFile(ctx)
	if err != nil {
		log.Debugf(ctx, "cli version check: %v", err)
		return
	}
	if !upgradecheck.Stale(cacheFile, time.Now()) {
		return
	}
	go func() {
		if err := upgradecheck.Refresh(ctx, cacheFile); err != nil {
			log.Debugf(ctx, "cli version check refresh failed: %v", err)
		}
	}()
}

// printUpgradeNotice prints a short advisory to stderr when a newer CLI release
// is available, based on the cached latest version. It is a no-op unless the
// check is enabled for this invocation. Callers must only invoke it after the
// command has succeeded, so an upgrade hint is never stacked on top of an error.
func printUpgradeNotice(ctx context.Context, cmd *cobra.Command) {
	if !upgradeCheckEnabled(ctx, cmd) {
		return
	}
	cacheFile, err := versionCheckCacheFile(ctx)
	if err != nil {
		return
	}
	current := build.GetInfo().Version
	latest, url, ok := upgradecheck.Outdated(cacheFile, current)
	if !ok {
		return
	}
	cmd.PrintErrln(cmdio.Yellow(ctx, upgradeNoticeMessage(current, latest, url)))
}

// upgradeNoticeMessage formats the advisory shown when a newer release exists.
// The leading blank line separates it from the command's own output.
func upgradeNoticeMessage(current, latest, url string) string {
	return fmt.Sprintf("\nA new release of the Databricks CLI is available: %s → %s\n%s",
		trimV(current), trimV(latest), url)
}

// upgradeCheckEnabled reports whether the outdated-version check should run for
// this invocation. It is suppressed in every context where the notice would be
// noise or where the user cannot act on it.
func upgradeCheckEnabled(ctx context.Context, cmd *cobra.Command) bool {
	// Released builds only; there is nothing to upgrade a dev/snapshot build to.
	// Returning early also avoids inspecting command flags for dev builds.
	if !upgradecheck.IsReleaseVersion(build.GetInfo().Version) {
		return false
	}
	return shouldNotify(notifyConditions{
		onRuntime:  dbr.RunsOnRuntime(ctx),
		textOutput: OutputType(cmd) == flags.OutputText,
		isTTY:      cmdio.IsOutputTTY(cmd.ErrOrStderr()),
		// Explicit guard for CI runners that allocate a TTY. Virtually all CI
		// providers set CI; see https://docs.github.com/actions/learn-github-actions/variables
		isCI: env.Get(ctx, "CI") != "",
	})
}

// notifyConditions captures the environment inputs that decide whether the
// outdated-version notice should run for a released build.
type notifyConditions struct {
	onRuntime  bool // on Databricks Runtime: version is platform-managed
	textOutput bool // machine-readable output is kept clean for scripts
	isTTY      bool // interactive terminal: also excludes pipes/redirects/most CI
	isCI       bool // continuous integration
}

// shouldNotify reports whether to run the version check for the given conditions.
func shouldNotify(c notifyConditions) bool {
	return c.textOutput && c.isTTY && !c.onRuntime && !c.isCI
}

func versionCheckCacheFile(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", versionCheckCacheName), nil
}

func trimV(version string) string {
	if len(version) > 0 && version[0] == 'v' {
		return version[1:]
	}
	return version
}
