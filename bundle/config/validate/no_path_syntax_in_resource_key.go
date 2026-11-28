package validate

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type noPathSyntaxInResourceKey struct{ bundle.RO }

// NoPathSyntaxInResourceKey rejects resource keys that contain "." or "[".
// The deployment engine identifies a resource by a path string such as
// "resources.jobs.<key>" and parses it back into components, so such a key
// resolves to a different resource path.
func NoPathSyntaxInResourceKey() bundle.ReadOnlyMutator {
	return &noPathSyntaxInResourceKey{}
}

func (m *noPathSyntaxInResourceKey) Name() string {
	return "validate:no_path_syntax_in_resource_key"
}

func (m *noPathSyntaxInResourceKey) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			key := p[2].Key()
			if strings.ContainsAny(key, ".[") {
				diags = append(diags, diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("resource key %q must not contain \".\" or \"[\"", key),
					Locations: v.Locations(),
					Paths:     dyn.ToStructPaths(p),
				})
			}
			return v, nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	// Typed-to-dynamic conversion iterates Go maps, so resource order is random; sort for stable output.
	slices.SortFunc(diags, func(x, y diag.Diagnostic) int {
		return cmp.Compare(x.Paths[0].String(), y.Paths[0].String())
	})

	return diags
}
