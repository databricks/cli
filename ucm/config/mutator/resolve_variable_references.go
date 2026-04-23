package mutator

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/ucm"
)

// maxResolutionRounds caps the fixed-point iteration used when a resolved
// value itself contains a reference. Matches bundle's budget; past 11 rounds
// the cost grows exponentially with pathological input.
const maxResolutionRounds = 11

// ResolveResourceReferences substitutes any ${resources.<kind>.<key>.<field>}
// reference in the loaded ucm config with the pointed-at value from the same
// tree. Non-resource references (var.*, workspace.*, anything else) are left
// untouched for later passes.
func ResolveResourceReferences() ucm.Mutator { return &resolveResourceReferences{} }

type resolveResourceReferences struct{}

func (m *resolveResourceReferences) Name() string { return "ResolveResourceReferences" }

func (m *resolveResourceReferences) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	err := u.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dynvar.Resolve(root, func(p dyn.Path) (dyn.Value, error) {
			if len(p) == 0 || p[0].Key() != "resources" {
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			}
			v, err := dyn.GetByPath(root, p)
			if err != nil {
				return dyn.InvalidValue, err
			}
			return v, nil
		})
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
	}
	return diags
}

// ResolveVariableReferences substitutes ${var.<name>} tokens (shorthand for
// ${variables.<name>.value}) everywhere they appear. Iterates up to
// maxResolutionRounds times to resolve references that themselves point at
// other references.
func ResolveVariableReferences() ucm.Mutator { return &resolveVariableReferences{} }

type resolveVariableReferences struct{}

func (m *resolveVariableReferences) Name() string { return "ResolveVariableReferences" }

func (m *resolveVariableReferences) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	varPath := dyn.NewPath(dyn.Key("var"))
	variablesKey := dyn.NewPath(dyn.Key("variables"))

	for round := range maxResolutionRounds {
		updates := false
		err := u.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
			return dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
				// Rewrite ${var.foo} into ${variables.foo.value}.
				resolved := path
				if resolved.HasPrefix(varPath) {
					np := dyn.NewPath(dyn.Key("variables"), resolved[1], dyn.Key("value"))
					if len(resolved) > 2 {
						np = np.Append(resolved[2:]...)
					}
					resolved = np
				}

				if !resolved.HasPrefix(variablesKey) {
					return dyn.InvalidValue, dynvar.ErrSkipResolution
				}

				v, err := dyn.GetByPath(root, resolved)
				if err != nil {
					return dyn.InvalidValue, err
				}
				if v.IsValid() {
					updates = true
				}
				return v, nil
			})
		})
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  err.Error(),
			})
			return diags
		}
		if !updates {
			break
		}
		if round == maxResolutionRounds-1 {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Variable references are too deep; stopping resolution. Unresolved variables may remain.",
			})
		}
	}
	return diags
}
