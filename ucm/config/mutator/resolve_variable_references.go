package mutator

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/ucm"
)

// ResolveResourceReferences returns a mutator that substitutes any
// ${resources.<kind>.<key>.<field>} reference in the loaded ucm config with
// the pointed-at value from the same tree. Non-resource references (var.*,
// workspace.*, anything else) are left untouched for later passes to handle.
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
