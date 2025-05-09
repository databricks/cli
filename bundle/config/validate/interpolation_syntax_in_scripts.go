package validate

import (
	"context"
	"fmt"
	"regexp"

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
	for k, script := range b.Config.Scripts {
		if re.MatchString(script.Content) {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Interpolation syntax ${...} in scripts is not allowed",
				Detail: `We do not support the ${...} interpolation syntax in scripts because
it's ambiguous whether it's a variable reference or reference to an
environment variable.`,
				Locations: b.Config.Value().Get(fmt.Sprintf("scripts.%s.content", k)).Locations(),
				Paths:     []dyn.Path{dyn.NewPath(dyn.Key("scripts"), dyn.Key(k), dyn.Key("content"))},
			})
		}
	}

	return diags
}
