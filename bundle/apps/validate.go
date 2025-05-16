package apps

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validate struct{}

func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	usedSourceCodePaths := make(map[string]string)

	for key, app := range b.Config.Resources.Apps {
		if app.SourceCodePath == "" {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Missing app source code path",
				Detail:    fmt.Sprintf("app resource '%s' is missing required source_code_path field", key),
				Locations: b.Config.GetLocations("resources.apps." + key),
			})
			continue
		}

		if _, ok := usedSourceCodePaths[app.SourceCodePath]; ok {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Duplicate app source code path",
				Detail:    fmt.Sprintf("app resource '%s' has the same source code path as app resource '%s', this will lead to the app configuration being overriden by each other", key, usedSourceCodePaths[app.SourceCodePath]),
				Locations: b.Config.GetLocations(fmt.Sprintf("resources.apps.%s.source_code_path", key)),
			})
		}
		usedSourceCodePaths[app.SourceCodePath] = key

		if app.Config != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "App config section detected",
				Detail:   fmt.Sprintf("remove 'config' from app resource '%s' section and use app.yml file in the root of this app instead", key),
			})
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
