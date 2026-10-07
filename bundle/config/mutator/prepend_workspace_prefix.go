package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
	patterns := []*structpath.PatternNode{
		structpath.MustParsePattern("workspace.root_path"),
		structpath.MustParsePattern("workspace.file_path"),
		structpath.MustParsePattern("workspace.artifact_path"),
		structpath.MustParsePattern("workspace.state_path"),
		structpath.MustParsePattern("workspace.resource_path"),
	}

	for _, pattern := range patterns {
		err := structvar.ForEach(b.Config.View(), pattern, func(p *structpath.PathNode, pv structvar.View) error {
			path, ok := pv.AsString()
			if !ok {
				return fmt.Errorf("expected string, got %s", pv.Kind())
			}

			// Skip prefixing if the path does not start with /, it might be variable reference or smth else.
			if !strings.HasPrefix(path, "/") {
				return nil
			}

			for _, prefix := range skipPrefixes {
				if strings.HasPrefix(path, prefix) {
					return nil
				}
			}

			// Set keeps the locations of the value, so diagnostics point at the original config line.
			return b.Config.Set(p, "/Workspace"+path)
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
