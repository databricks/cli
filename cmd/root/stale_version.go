package root

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/versioncheck"
)

const (
	staleVersionThreshold = 6 * 30 * 24 * time.Hour
	installDocsURL        = "https://docs.databricks.com/dev-tools/cli/install.html"
)

// staleVersionWarning returns the warning for a build older than the threshold, or "" otherwise.
// An empty upgradeCommand falls back to the install docs.
func staleVersionWarning(buildTime, now time.Time, upgradeCommand string) string {
	// Builds without an embedded timestamp report the Unix epoch.
	if buildTime.Unix() <= 0 || now.Sub(buildTime) < staleVersionThreshold {
		return ""
	}
	msg := fmt.Sprintf("Warning: this version of the Databricks CLI was built on %s and is more than 6 months old. "+
		"We strongly recommend updating to the latest version.\n", buildTime.UTC().Format(time.DateOnly))
	if upgradeCommand != "" {
		return msg + "To upgrade, run: " + upgradeCommand + "\n"
	}
	return msg + "See " + installDocsURL + " to upgrade.\n"
}

func warnIfStaleVersion(ctx context.Context) {
	info := build.GetInfo()
	if info.IsDevelopment() {
		return
	}
	// DBR installs the latest CLI by default, so skip the warning there.
	if dbr.HasDetection(ctx) && dbr.RunsOnRuntime(ctx) {
		return
	}
	_, command := versioncheck.DetectInstallMethod(ctx)
	if msg := staleVersionWarning(info.BuildTime, time.Now(), command); msg != "" {
		cmdio.LogString(ctx, msg)
	}
}
