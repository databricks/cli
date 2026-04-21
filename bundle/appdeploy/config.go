package appdeploy

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// ResolveAppConfig returns the app config with `${resources.*}` variable references
// resolved against the current bundle state. The app runtime configuration (env vars,
// command) can reference other bundle resources whose properties are known only after
// the initialization phase — so this has to be called after resources have been
// created and the bundle Config reflects their current state.
//
// appKey is the map key under `resources.apps` (usually equal to `app.Name` but not
// guaranteed — resources may be keyed by a local alias).
//
// The function takes a pointer to config.Root rather than *bundle.Bundle to avoid an
// import cycle between bundle and appdeploy (bundle → direct → dresources → appdeploy).
func ResolveAppConfig(cfg *config.Root, appKey string, app *resources.App) (*resources.AppConfig, error) {
	if app == nil || app.Config == nil {
		return nil, nil
	}

	root := cfg.Value()

	// Normalize the full config so that all typed fields are present, even those
	// not explicitly set. This allows looking up resource properties by path.
	normalized, _ := convert.Normalize(cfg, root, convert.IncludeMissingFields)

	configPath := dyn.MustPathFromString("resources.apps." + appKey + ".config")
	configV, err := dyn.GetByPath(root, configPath)
	if err != nil || !configV.IsValid() {
		return app.Config, nil //nolint:nilerr // missing config path means use default config
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

	var resolvedConfig resources.AppConfig
	if err := convert.ToTyped(&resolvedConfig, resolved); err != nil {
		return nil, err
	}
	return &resolvedConfig, nil
}
