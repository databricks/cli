package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
		contentPath := structpath.NewPath(nil, "scripts", k, "content")

		if script.Content == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Script %s has no content", k),
				Paths:    []*structpath.PathNode{contentPath},
			})
			continue
		}

		diags = diags.Extend(validateScriptContent(b, k, script.Content, contentPath))
		diags = diags.Extend(validateScriptEnv(b, k, script.Env))
	}

	return diags
}

// validateScriptContent rejects any ${...} reference in a script's content.
// Content is passed to the shell as-is, so ${...} would be ambiguous with a
// bundle reference; reference a declared env entry with $NAME instead.
func validateScriptContent(b *bundle.Bundle, key, content string, p *structpath.PathNode) diag.Diagnostics {
	ref, ok := structvar.NewRef(content)
	if !ok {
		return nil
	}

	first := ref.Matches[0][0]
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("Found %s in script %s.content. Interpolation syntax ${...} is not supported in script content", first, key),
		Detail: `The ${...} syntax is not allowed in script content. To use a bundle value,
declare an environment variable in the script's "env:" section and reference it
from "content" with $NAME:

  scripts:
    ` + key + `:
      env:
        MY_VAR: ${var.foo}
      content: echo "$MY_VAR"`,
		Paths: []*structpath.PathNode{p},
	}}
}

func validateScriptEnv(b *bundle.Bundle, key string, env map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, name := range slices.Sorted(maps.Keys(env)) {
		ref, ok := structvar.NewRef(env[name])
		if !ok {
			continue
		}

		envValuePath := structpath.NewPath(nil, "scripts", key, "env", name)

		for _, refPath := range ref.References() {
			prefix, _, _ := strings.Cut(refPath, ".")
			if slices.Contains(allowedEnvRefPrefixes, prefix) {
				continue
			}
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("${%s} cannot be used in scripts.%s.env.%s; only ${bundle.*}, ${workspace.*}, and ${var.*} are resolved before scripts execute", refPath, key, name),
				Paths:    []*structpath.PathNode{envValuePath},
			})
		}
	}

	return diags
}
