package validate

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type noInterpolationSyntaxInScripts struct{}

func NoInterpolationSyntaxInScripts() bundle.Mutator {
	return &noInterpolationSyntaxInScripts{}
}

func (f *noInterpolationSyntaxInScripts) Name() string {
	return "validate:no_interpolation_syntax_in_scripts"
}

func (f *noInterpolationSyntaxInScripts) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	re := regexp.MustCompile(`\$\{.*\}`)

	// Sort the scripts to have a deterministic order for the
	// generated diagnostics.
	var scriptKeys []string
	for k := range b.Config.Scripts {
		scriptKeys = append(scriptKeys, k)
	}
	sort.Strings(scriptKeys)

	for _, k := range scriptKeys {
		script := b.Config.Scripts[k]
		match := re.FindString(script.Content)
		if match == "" {
			continue
		}

		p := dyn.NewPath(dyn.Key("scripts"), dyn.Key(k), dyn.Key("content"))
		v, err := dyn.GetByPath(b.Config.Value(), p)
		if err != nil {
			return diag.FromErr(err)
		}

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

	return diags
}
