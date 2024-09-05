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
	paths := []string{
		"/Workspace/${workspace.root_path}",
		"/Workspace/${workspace.file_path}",
		"/Workspace/${workspace.artifact_path}",
		"/Workspace/${workspace.state_path}",
	}

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		// Walk through the bundle configuration, check all the string leafs and
		// see if any of the prefixes are used in the remote path.
		return dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			vv, ok := v.AsString()
			if !ok {
				return v, nil
			}

			for _, path := range paths {
				if strings.Contains(vv, path) {
					diags = append(diags, diag.Diagnostic{
						Severity:  diag.Warning,
						Summary:   fmt.Sprintf("%s used in the remote path %s. Please change to use %s instead", path, vv, strings.ReplaceAll(vv, "/Workspace/", "")),
						Locations: v.Locations(),
						Paths:     []dyn.Path{p},
					})

					// Remove the workspace prefix from the string.
					return dyn.NewValue(strings.ReplaceAll(vv, "/Workspace/", ""), v.Locations()), nil
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
