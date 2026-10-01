package root

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
)

const staleVersionThreshold = 6 * 30 * 24 * time.Hour

func staleVersionWarning(buildTime, now time.Time) string {
	// Builds without an embedded timestamp (e.g. local `go build`) report the Unix epoch.
	if buildTime.Unix() <= 0 || now.Sub(buildTime) < staleVersionThreshold {
		return ""
	}
	return fmt.Sprintf("Warning: this version of the Databricks CLI was built on %s and is more than 6 months old. "+
		"We strongly recommend updating to the latest version: https://docs.databricks.com/dev-tools/cli/install.html\n",
		buildTime.UTC().Format(time.DateOnly))
}

func warnIfStaleVersion(ctx context.Context) {
	// DBR installs the latest CLI by default, so skip the warning there.
	if dbr.HasDetection(ctx) && dbr.RunsOnRuntime(ctx) {
		return
	}
	if msg := staleVersionWarning(build.GetInfo().BuildTime, time.Now()); msg != "" {
		cmdio.LogString(ctx, msg)
	}
}
