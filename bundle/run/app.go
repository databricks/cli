package run

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

func logProgress(ctx context.Context, msg string) {
	if msg == "" {
		return
	}
	cmdio.LogString(ctx, "✓ "+msg)
}

type appRunner struct {
	key

	bundle *bundle.Bundle
	app    *resources.App
}

func (a *appRunner) Name() string {
	if a.app == nil {
		return ""
	}

	return a.app.Name
}

func isAppComputeStopped(app *apps.App) bool {
	return app.ComputeStatus == nil ||
		(app.ComputeStatus.State == apps.ComputeStateStopped || app.ComputeStatus.State == apps.ComputeStateError)
}

func (a *appRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	app := a.app
	b := a.bundle
	if app == nil {
		return nil, errors.New("app is not defined")
	}

	logProgress(ctx, "Getting the status of the app "+app.Name)
	w := b.WorkspaceClient()

	// Check the status of the app first.
	createdApp, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: app.Name})
	if err != nil {
		return nil, err
	}

	if createdApp.AppStatus != nil {
		logProgress(ctx, fmt.Sprintf("App is in %s state", createdApp.AppStatus.State))
	}

	if createdApp.ComputeStatus != nil {
		logProgress(ctx, fmt.Sprintf("App compute is in %s state", createdApp.ComputeStatus.State))
	}

	// There could be 2 reasons why the app is not running:
	// 1. The app is new and was never deployed yet.
	// 2. The app was stopped (compute not running).
	// We need to start the app only if the compute is not running.
	if isAppComputeStopped(createdApp) {
		err := a.start(ctx)
		if err != nil {
			return nil, err
		}
	}

	// If the app is starting, we need to wait for it to be active before we can deploy it.
	if isAppComputeStarting(createdApp) {
		_, err := w.Apps.WaitGetAppActive(ctx, app.Name, 20*time.Minute, nil)
		if err != nil {
			return nil, err
		}
	}

	// Deploy the app.
	err = a.deploy(ctx)
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "You can access the app at "+createdApp.Url)
	return nil, nil
}

func isAppComputeStarting(app *apps.App) bool {
	return app.ComputeStatus != nil && app.ComputeStatus.State == apps.ComputeStateStarting
}

func (a *appRunner) start(ctx context.Context) error {
	app := a.app
	b := a.bundle
	w := b.WorkspaceClient()

	logProgress(ctx, "Starting the app "+app.Name)
	wait, err := w.Apps.Start(ctx, apps.StartAppRequest{Name: app.Name})
	if err != nil {
		return err
	}

	startedApp, err := wait.OnProgress(func(p *apps.App) {
		if p.AppStatus == nil {
			return
		}
		logProgress(ctx, "App is starting...")
	}).Get()
	if err != nil {
		return err
	}

	// After the app is started (meaning the compute is running), the API will return the app object with the
	// active and pending deployments fields (if any). If there are active or pending deployments,
	// we need to wait for them to complete before we can do the new deployment.
	// Otherwise, the new deployment will fail.
	err = appdeploy.WaitForDeploymentToComplete(ctx, w, startedApp)
	if err != nil {
		return err
	}

	logProgress(ctx, "App is started!")
	return nil
}

func (a *appRunner) deploy(ctx context.Context) error {
	w := a.bundle.WorkspaceClient()
	config, err := a.resolvedConfig()
	if err != nil {
		return err
	}
	deployment := appdeploy.BuildDeployment(a.app.SourceCodePath, config, a.app.GitSource)
	return appdeploy.Deploy(ctx, w, a.app.Name, deployment)
}

// resolvedConfig returns the app config with any ${resources.*} variable references
// resolved against the current bundle state. This is needed because the app runtime
// configuration (env vars, command) can reference other bundle resources whose
// properties are known only after the initialization phase.
func (a *appRunner) resolvedConfig() (*resources.AppConfig, error) {
	if a.app.Config == nil {
		return nil, nil
	}

	root := a.bundle.Config.Value()

	// Normalize the full config so that all typed fields are present, even those
	// not explicitly set. This allows looking up resource properties by path.
	normalized, _ := convert.Normalize(a.bundle.Config, root, convert.IncludeMissingFields)

	// Get the app's config section as a dyn.Value to resolve references in it.
	// The key is of the form "apps.<name>", so the full path is "resources.apps.<name>.config".
	configPath := dyn.MustPathFromString("resources." + a.Key() + ".config")
	configV, err := dyn.GetByPath(root, configPath)
	if err != nil || !configV.IsValid() { //nolint:nilerr // missing config path means use default config
		return a.app.Config, nil
	}

	resourcesPrefix := dyn.MustPathFromString("resources")

	// Resolve ${resources.*} references in the app config against the full bundle config.
	// Other variable types (bundle.*, workspace.*, variables.*) are already resolved
	// during the initialization phase and are left in place if encountered here.
	resolved, err := dynvar.Resolve(configV, func(path dyn.Path) (dyn.Value, error) {
		if !path.HasPrefix(resourcesPrefix) {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}
		return dyn.GetByPath(normalized, path)
	})
	if err != nil {
		return nil, err
	}

	var config resources.AppConfig
	if err := convert.ToTyped(&config, resolved); err != nil {
		return nil, err
	}
	return &config, nil
}

func (a *appRunner) Cancel(ctx context.Context) error {
	// We should cancel the app by stopping it.
	app := a.app
	b := a.bundle
	if app == nil {
		return errors.New("app is not defined")
	}

	w := b.WorkspaceClient()

	logProgress(ctx, "Stopping app "+app.Name)
	wait, err := w.Apps.Stop(ctx, apps.StopAppRequest{Name: app.Name})
	if err != nil {
		return err
	}

	_, err = wait.OnProgress(func(p *apps.App) {
		if p.AppStatus == nil {
			return
		}
		logProgress(ctx, p.AppStatus.Message)
	}).Get()

	logProgress(ctx, "App is stopped!")
	return err
}

func (a *appRunner) Restart(ctx context.Context, opts *Options) (output.RunOutput, error) {
	// We should restart the app by just running it again meaning a new app deployment will be done.
	return a.Run(ctx, opts)
}

func (a *appRunner) ParseArgs(args []string, opts *Options) error {
	if len(args) == 0 {
		return nil
	}

	return fmt.Errorf("received %d unexpected positional arguments", len(args))
}

func (a *appRunner) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
