package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type noWorkspacePrefixUsed struct{}

// NoWorkspacePrefixUsed ensures that no workspace prefix plus workspace path variable is used in the remote path.
func NoWorkspacePrefixUsed() bundle.Mutator {
	return &noWorkspacePrefixUsed{}
}

func (m *noWorkspacePrefixUsed) Name() string {
	return "validate:no_workspace_prefix_used"
}

func (m *noWorkspacePrefixUsed) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}
	paths := []string{
		"/Workspace/${workspace.root_path}",
		"/Workspace/${workspace.file_path}",
		"/Workspace/${workspace.artifact_path}",
		"/Workspace/${workspace.state_path}",
	}

	// Walk through the bundle configuration, check all the string leafs and
	// see if any of the prefixes are used in the remote path.
	_, err := dyn.Walk(b.Config.Value(), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		vv, ok := v.AsString()
		if !ok {
			return v, nil
		}

		for _, path := range paths {
			if strings.Contains(vv, path) {
				diags = append(diags, diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("%s used in the remote path %s. Please change to use %s instead", path, vv, strings.ReplaceAll(vv, "/Workspace/", "")),
					Locations: v.Locations(),
					Paths:     []dyn.Path{p},
				})
			}
		}

		return v, nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
