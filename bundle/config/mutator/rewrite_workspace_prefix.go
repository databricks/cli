package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type rewriteWorkspacePrefix struct{}

// RewriteWorkspacePrefix finds any strings in bundle configration that have
// workspace prefix plus workspace path variable used and removes workspace prefix from it.
func RewriteWorkspacePrefix() bundle.Mutator {
	return &rewriteWorkspacePrefix{}
}

func (m *rewriteWorkspacePrefix) Name() string {
	return "RewriteWorkspacePrefix"
}

func (m *rewriteWorkspacePrefix) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}
	paths := map[string]string{
		"/Workspace/${workspace.root_path}":     "${workspace.root_path}",
		"/Workspace${workspace.root_path}":      "${workspace.root_path}",
		"/Workspace/${workspace.file_path}":     "${workspace.file_path}",
		"/Workspace${workspace.file_path}":      "${workspace.file_path}",
		"/Workspace/${workspace.artifact_path}": "${workspace.artifact_path}",
		"/Workspace${workspace.artifact_path}":  "${workspace.artifact_path}",
		"/Workspace/${workspace.state_path}":    "${workspace.state_path}",
		"/Workspace${workspace.state_path}":     "${workspace.state_path}",
	}

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		// Walk through the bundle configuration, check all the string leafs and
		// see if any of the prefixes are used in the remote path.
		return dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			vv, ok := v.AsString()
			if !ok {
				return v, nil
			}

			for path, replacePath := range paths {
				if strings.Contains(vv, path) {
					newPath := strings.Replace(vv, path, replacePath, 1)
					diags = append(diags, diag.Diagnostic{
						Severity:  diag.Warning,
						Summary:   fmt.Sprintf("substring %q found in %q. Please update this to %q.", path, vv, newPath),
						Detail:    "For more information, please refer to: https://docs.databricks.com/en/release-notes/dev-tools/bundles.html#workspace-paths",
						Locations: v.Locations(),
						Paths:     []dyn.Path{p},
					})

					// Remove the workspace prefix from the string.
					return dyn.NewValue(newPath, v.Locations()), nil
				}
			}

			return v, nil
		})
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
