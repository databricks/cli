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

// RewriteWorkspacePrefix finds any strings in bundle configuration that have
// workspace prefix plus workspace path variable used and removes workspace prefix from it.
func RewriteWorkspacePrefix() bundle.Mutator {
	return &rewriteWorkspacePrefix{}
}

func (m *rewriteWorkspacePrefix) Name() string {
	return "RewriteWorkspacePrefix"
}

func (m *rewriteWorkspacePrefix) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}
	// The patterns must cover every path that PrependWorkspacePrefix prefixes,
	// otherwise interpolation produces a doubled "/Workspace/Workspace/..." path.
	// A slice (not a map) keeps the rewrite and warning order deterministic when
	// multiple patterns occur in the same string.
	paths := []struct {
		pattern     string
		replacement string
	}{
		{"/Workspace/${workspace.root_path}", "${workspace.root_path}"},
		{"/Workspace${workspace.root_path}", "${workspace.root_path}"},
		{"/Workspace/${workspace.file_path}", "${workspace.file_path}"},
		{"/Workspace${workspace.file_path}", "${workspace.file_path}"},
		{"/Workspace/${workspace.artifact_path}", "${workspace.artifact_path}"},
		{"/Workspace${workspace.artifact_path}", "${workspace.artifact_path}"},
		{"/Workspace/${workspace.state_path}", "${workspace.state_path}"},
		{"/Workspace${workspace.state_path}", "${workspace.state_path}"},
		{"/Workspace/${workspace.resource_path}", "${workspace.resource_path}"},
		{"/Workspace${workspace.resource_path}", "${workspace.resource_path}"},
	}

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		// Walk through the bundle configuration, check all the string leafs and
		// see if any of the prefixes are used in the remote path.
		return dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			vv, ok := v.AsString()
			if !ok {
				return v, nil
			}

			newPath := vv
			for _, rewrite := range paths {
				if !strings.Contains(newPath, rewrite.pattern) {
					continue
				}

				newPath = strings.ReplaceAll(newPath, rewrite.pattern, rewrite.replacement)
				diags = append(diags, diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   fmt.Sprintf("substring %q found in %q. Please update this to %q.", rewrite.pattern, vv, newPath),
					Detail:    "For more information, please refer to: https://docs.databricks.com/en/release-notes/dev-tools/bundles.html#workspace-paths",
					Locations: v.Locations(),
					Paths:     []dyn.Path{p},
				})
			}

			if newPath == vv {
				return v, nil
			}

			// Remove the workspace prefix from the string.
			return dyn.NewValue(newPath, v.Locations()), nil
		})
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
