package apps

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validate struct{}

func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	possibleConfigFiles := []string{"app.yml", "app.yaml"}
	usedSourceCodePaths := make(map[string]string)

	for key, app := range b.Config.Resources.Apps {
		if _, ok := usedSourceCodePaths[app.SourceCodePath]; ok {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Duplicate app source code path",
				Detail:    fmt.Sprintf("app resource '%s' has the same source code path as app resource '%s', this will lead to the app configuration being overriden by each other", key, usedSourceCodePaths[app.SourceCodePath]),
				Locations: b.Config.GetLocations(fmt.Sprintf("resources.apps.%s.source_code_path", key)),
			})
		}
		usedSourceCodePaths[app.SourceCodePath] = key

		for _, configFile := range possibleConfigFiles {
			appPath := strings.TrimPrefix(app.SourceCodePath, b.Config.Workspace.FilePath)
			cf := path.Join(appPath, configFile)
			if _, err := b.SyncRoot.Stat(cf); err == nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  configFile + " detected",
					Detail:   fmt.Sprintf("remove %s and use 'config' property for app resource '%s' instead", cf, app.Name),
				})
			}
		}
	}

	return diags
}

func (v *validate) Name() string {
	return "apps.Validate"
}

func Validate() bundle.Mutator {
	return &validate{}
}
