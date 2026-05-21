package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type validateScripts struct{}

func Scripts() bundle.Mutator {
	return &validateScripts{}
}

func (f *validateScripts) Name() string {
	return "validate:scripts"
}

// allowedEnvRefPrefixes are the variable prefixes that may appear in a
// script's "env:" section. These match the prefixes resolved before scripts
// execute (defaultPrefixes in resolve_variable_references.go); "var" is the
// shorthand for "variables".
var allowedEnvRefPrefixes = []string{"bundle", "workspace", "var", "variables"}

func (f *validateScripts) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Sort the scripts to have a deterministic order for the
	// generated diagnostics.
	scriptKeys := slices.Sorted(maps.Keys(b.Config.Scripts))

	for _, k := range scriptKeys {
		script := b.Config.Scripts[k]
		contentPath := dyn.NewPath(dyn.Key("scripts"), dyn.Key(k), dyn.Key("content"))

		if script.Content == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Script %s has no content", k),
				Paths:    []dyn.Path{contentPath},
			})
			continue
		}

		diags = diags.Extend(validateScriptContent(b, k, script.Content, contentPath))
		diags = diags.Extend(validateScriptEnv(b, k, script.Env))
	}

	return diags
}

func validateScriptContent(b *bundle.Bundle, key, content string, p dyn.Path) diag.Diagnostics {
	ref, ok := dynvar.NewRef(dyn.V(content))
	if !ok {
		return nil
	}

	first := ref.Matches[0][0]
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("Found %s in script %s.content. Interpolation syntax ${...} is not supported in script content", first, key),
		Detail: `Script content is passed to the shell as-is, so ${...} is left for the shell to expand.
To interpolate a bundle value into the script, declare an environment variable
in the script's "env:" section and reference it from "content" with $NAME:

  scripts:
    ` + key + `:
      env:
        MY_VAR: ${var.foo}
      content: echo "$MY_VAR"`,
		Locations: locationsForPath(b, p),
		Paths:     []dyn.Path{p},
	}}
}

func validateScriptEnv(b *bundle.Bundle, key string, env map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, name := range slices.Sorted(maps.Keys(env)) {
		ref, ok := dynvar.NewRef(dyn.V(env[name]))
		if !ok {
			continue
		}

		envValuePath := dyn.NewPath(dyn.Key("scripts"), dyn.Key(key), dyn.Key("env"), dyn.Key(name))

		for _, refPath := range ref.References() {
			prefix, _, _ := strings.Cut(refPath, ".")
			if slices.Contains(allowedEnvRefPrefixes, prefix) {
				continue
			}
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("${%s} cannot be used in scripts.%s.env.%s; only ${bundle.*}, ${workspace.*}, and ${var.*} are resolved before scripts execute", refPath, key, name),
				Locations: locationsForPath(b, envValuePath),
				Paths:     []dyn.Path{envValuePath},
			})
		}
	}

	return diags
}

func locationsForPath(b *bundle.Bundle, p dyn.Path) []dyn.Location {
	v, err := dyn.GetByPath(b.Config.Value(), p)
	if err != nil {
		return nil
	}
	return v.Locations()
}
