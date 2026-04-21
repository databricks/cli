package phases

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// DeployApps uploads source code and triggers an AppDeployment for every app in the
// bundle. It is called at the end of the Deploy phase when the user asks for a
// single-command deploy-and-push (via `--deploy-apps` or `b.DeployApps = true`).
//
// The resource CRUD path (terraform/direct) provisions the app compute but does not
// push source code, which is why this exists as a separate phase: `w.Apps.Deploy`
// and file upload to the workspace are not modelled as resources.
func DeployApps(ctx context.Context, b *bundle.Bundle) {
	appCount := len(b.Config.Resources.Apps)
	if !b.DeployApps {
		if appCount > 0 {
			cmdio.LogString(ctx, skippedAppsMessage(b))
		}
		return
	}
	if appCount == 0 {
		return
	}

	log.Info(ctx, "Phase: deploy apps")
	cmdio.LogString(ctx, "Deploying app source code...")

	w := b.WorkspaceClient(ctx)
	var failures []error
	for key, app := range b.Config.Resources.Apps {
		if app == nil {
			continue
		}
		cmdio.LogString(ctx, fmt.Sprintf("✓ Deploying app source for %s", app.Name))

		config, err := appdeploy.ResolveAppConfig(&b.Config, key, app)
		if err != nil {
			failures = append(failures, fmt.Errorf("app %s: resolve config: %w", key, err))
			continue
		}

		deployment := appdeploy.BuildDeployment(app.SourceCodePath, config, app.GitSource)
		if err := appdeploy.Deploy(ctx, w, app.Name, deployment); err != nil {
			failures = append(failures, fmt.Errorf("app %s: %w", key, err))
			continue
		}
	}

	if len(failures) > 0 {
		logdiag.LogError(ctx, errors.Join(failures...))
		return
	}
	cmdio.LogString(ctx, "App source code deployed!")
}

func skippedAppsMessage(b *bundle.Bundle) string {
	keys := make([]string, 0, len(b.Config.Resources.Apps))
	for key := range b.Config.Resources.Apps {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var lines []string
	lines = append(lines, fmt.Sprintf("Bundle contains %d Apps, but --deploy-apps was not set, not deploying apps. To deploy, run:", len(keys)))
	for _, key := range keys {
		lines = append(lines, "  databricks bundle run "+key)
	}
	return strings.Join(lines, "\n")
}
