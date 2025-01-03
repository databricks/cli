package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type prependWorkspacePrefix struct{}

// PrependWorkspacePrefix prepends the workspace root path to all paths in the bundle.
func PrependWorkspacePrefix() bundle.Mutator {
	return &prependWorkspacePrefix{}
}

func (m *prependWorkspacePrefix) Name() string {
	return "PrependWorkspacePrefix"
}

var skipPrefixes = []string{
	"/Workspace/",
	"/Volumes/",
}

func (m *prependWorkspacePrefix) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	patterns := []dyn.Pattern{
		dyn.NewPattern(dyn.Key("workspace"), dyn.Key("root_path")),
		dyn.NewPattern(dyn.Key("workspace"), dyn.Key("file_path")),
		dyn.NewPattern(dyn.Key("workspace"), dyn.Key("artifact_path")),
		dyn.NewPattern(dyn.Key("workspace"), dyn.Key("state_path")),
		dyn.NewPattern(dyn.Key("workspace"), dyn.Key("resource_path")),
	}

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, pattern := range patterns {
			v, err = dyn.MapByPattern(v, pattern, func(p dyn.Path, pv dyn.Value) (dyn.Value, error) {
				path, ok := pv.AsString()
				if !ok {
					return dyn.InvalidValue, fmt.Errorf("expected string, got %s", v.Kind())
				}

				// Skip prefixing if the path does not start with /, it might be variable reference or smth else.
				if !strings.HasPrefix(path, "/") {
					return pv, nil
				}

				for _, prefix := range skipPrefixes {
					if strings.HasPrefix(path, prefix) {
						return pv, nil
					}
				}

				return dyn.NewValue("/Workspace"+path, v.Locations()), nil
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
