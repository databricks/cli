package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type danglingResourceReferences struct{}

// DanglingResourceReferences rejects ${resources.*} references whose target
// resource is not defined in the bundle. Deploy fails later with an invalid
// dependency (direct) or an unresolved reference (terraform); catch it here.
func DanglingResourceReferences() bundle.Mutator {
	return &danglingResourceReferences{}
}

func (m *danglingResourceReferences) Name() string {
	return "validate:dangling_resource_references"
}

func (m *danglingResourceReferences) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	_ = dyn.WalkReadOnly(b.Config.Value(), func(path dyn.Path, v dyn.Value) error {
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			if !strings.HasPrefix(r, "resources.") {
				continue
			}
			if d := checkDanglingResourceReference(b, r, path, v.Locations()); d != nil {
				diags = append(diags, *d)
			}
		}
		return nil
	})

	return diags
}

// checkDanglingResourceReference checks a reference like
// "resources.jobs.missing.id" and returns a diagnostic when the resource
// (resources.jobs.missing) is not defined.
func checkDanglingResourceReference(b *bundle.Bundle, ref string, path dyn.Path, locs []dyn.Location) *diag.Diagnostic {
	p, err := dyn.NewPathFromString(ref)
	// resources.<group>.<name>[.<field>...]
	if err != nil || len(p) < 3 || p[0].Key() != "resources" {
		return nil
	}

	// Identity is resources.<group>.<name>; trailing fields (.id, .permissions, …)
	// are resolved at deploy time and are not required to exist in config.
	resourceKey := p[:3]
	v, err := dyn.GetByPath(b.Config.Value(), resourceKey)
	if err == nil && v.Kind() != dyn.KindInvalid && v.Kind() != dyn.KindNil {
		return nil
	}

	d := &diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("reference does not exist: ${%s}", ref),
		Paths:    []dyn.Path{path},
	}
	// ApplyBundlePermissions rewrites permission entries without locations; skip
	// empty ones so we don't print "in :0:0".
	for _, loc := range locs {
		if loc.File != "" {
			d.Locations = append(d.Locations, loc)
		}
	}
	return d
}
