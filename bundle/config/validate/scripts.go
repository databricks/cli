package validate

import (
	"context"
	"fmt"
	"regexp"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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

	re := regexp.MustCompile(`\$\{.*\}`)

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

		// Check for interpolation syntax
		match := re.FindString(script.Content)
		if match != "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Found %s in script %s. Interpolation syntax ${...} is not allowed in scripts", match, k),
				Detail: `We do not support the ${...} interpolation syntax in scripts because
it's ambiguous whether it's a variable reference or reference to an
environment variable.`,
				Locations: v.Locations(),
				Paths:     []dyn.Path{p},
			})
		}
	}

	return diags
}
