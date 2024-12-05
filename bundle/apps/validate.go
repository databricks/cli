package apps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validate struct {
}

func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, app := range b.Config.Resources.Apps {
		possibleConfigFiles := []string{"app.yml", "app.yaml"}
		for _, configFile := range possibleConfigFiles {
			cf := filepath.Join(b.SyncRootPath, app.SourceCodePath, configFile)
			if _, err := os.Stat(cf); err == nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "app.yml detected",
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
