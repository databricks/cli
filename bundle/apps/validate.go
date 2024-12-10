package apps

import (
	"context"
	"fmt"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validate struct {
}

func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	possibleConfigFiles := []string{"app.yml", "app.yaml"}

	for _, app := range b.Config.Resources.Apps {
		for _, configFile := range possibleConfigFiles {
			cf := path.Join(app.SourceCodePath, configFile)
			if _, err := b.SyncRoot.Stat(cf); err == nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("%s detected", configFile),
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
