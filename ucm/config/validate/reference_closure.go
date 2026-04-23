package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/ucm"
)

// ReferenceClosure errors when any ${resources.<kind>.<key>.*} interpolation
// points at a resource the user did not declare.
//
// Safe to run before OR after ResolveResourceReferences: resolved references
// no longer match the pattern, and unresolvable ones stay in place. The
// terraform engine runs it before Build (so the TF JSON never ships broken
// refs); the direct engine runs it after ResolveResourceReferences (so any
// leftover ${resources.*} is guaranteed-dangling).
//
// Scoped to ${resources.*} tokens only. Non-resource references (${var.*},
// ${workspace.*}, etc.) are ignored here: the variable-resolution pass that
// handles them lands in M2/W1 and has its own closure check.
//
// TODO(M2/W1): extend to ${var.*} once variables are interpolated upstream.
func ReferenceClosure() ucm.Mutator { return &referenceClosure{} }

type referenceClosure struct{}

func (m *referenceClosure) Name() string { return "validate:reference_closure" }

const resourcesNamespace = "resources"

func (m *referenceClosure) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	var diags diag.Diagnostics
	root := u.Config.Value()

	err := dyn.WalkReadOnly(root, func(p dyn.Path, v dyn.Value) error {
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, target := range ref.References() {
			targetPath, parseErr := dyn.NewPathFromString(target)
			if parseErr != nil || !isResourceTarget(targetPath) {
				continue
			}
			if !resourceExists(root, targetPath) {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary: fmt.Sprintf(
						"unresolved reference ${%s} at %s: target resource is not declared in config",
						target, p.String(),
					),
					Paths:     []dyn.Path{p},
					Locations: locsOf(v),
				})
			}
		}
		return nil
	})
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

// isResourceTarget reports whether target is a ${resources.<kind>.<key>.*}
// reference that can be checked for closure.
func isResourceTarget(target dyn.Path) bool {
	return len(target) >= 3 && target[0].Key() == resourcesNamespace
}

// resourceExists returns true when resources.<kind>.<key> is present in root.
// We check the 3-segment prefix only — a reference to a sub-field of a
// declared resource is always closed (the field may just be empty).
func resourceExists(root dyn.Value, target dyn.Path) bool {
	prefix := target[:3]
	_, err := dyn.GetByPath(root, prefix)
	return err == nil
}

func locsOf(v dyn.Value) []dyn.Location {
	loc := v.Location()
	if loc.File == "" && loc.Line == 0 {
		return nil
	}
	return []dyn.Location{loc}
}
