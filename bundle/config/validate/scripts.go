package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/utils"
)

type validateScripts struct{}

func Scripts() bundle.Mutator {
	return &validateScripts{}
}

func (f *validateScripts) Name() string {
	return "validate:scripts"
}

func (f *validateScripts) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Sort the scripts to have a deterministic order for the
	// generated diagnostics.
	scriptKeys := utils.SortedKeys(b.Config.Scripts)

	for _, k := range scriptKeys {
		script := b.Config.Scripts[k]
		p := dyn.NewPath(dyn.Key("scripts"), dyn.Key(k), dyn.Key("content"))

		if script.Content == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Script %s has no content", k),
				Paths:    []dyn.Path{p},
			})
			continue
		}

		v, err := dyn.GetByPath(b.Config.Value(), p)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}

		// Find all interpolation references in the script content.
		// This uses the same regex as the variable resolver, so it only matches
		// patterns that look like DAB variable references (not bash parameter
		// expansion like ${VAR:-default}).
		refs := dynvar.FindAllInterpolationReferences(script.Content)
		for _, ref := range refs {
			// Check if this reference has a valid DAB prefix.
			// Valid prefixes are: var, bundle, workspace, variables, resources, artifacts
			if !dynvar.HasValidDABPrefix(ref.Path) {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Invalid interpolation reference %s in script %s", ref.Match, k),
					Detail: fmt.Sprintf(`The interpolation reference %s does not start with a valid prefix.
Valid prefixes are: %s.

If you meant to use an environment variable, use $%s instead of %s.
If you meant to use a bundle variable, use ${var.%s} instead.`,
						ref.Match,
						strings.Join(dynvar.ValidDABPrefixes, ", "),
						ref.Path,
						ref.Match,
						ref.Path,
					),
					Locations: v.Locations(),
					Paths:     []dyn.Path{p},
				})
			}
		}
	}

	return diags
}
