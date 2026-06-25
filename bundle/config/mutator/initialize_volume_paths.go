package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type initializeVolumePaths struct{}

// InitializeVolumePaths sets resources.volumes.*.volume_path from catalog, schema, and name.
//
// catalog_name and schema_name may be ${resources.catalogs/schemas.<key>.name} references
// (for example, after CaptureUCDependencies rewrites implicit dependencies). We resolve those
// references locally to compute the path, but we do not write the resolved values back: the
// original references are preserved so validate and plan keep showing them.
//
// A component that cannot be resolved locally (for example a remote field only known at plan or
// deploy time) is left as a ${...} reference and embedded into volume_path. The embedded
// reference is then carried through ${resources.volumes.<key>.volume_path} interpolation and
// resolved later by the engine during plan or deploy, the same way any other resource reference
// is. This enables ${resources.volumes.<key>.volume_path} interpolation during initialize.
func InitializeVolumePaths() bundle.Mutator {
	return &initializeVolumePaths{}
}

func (m *initializeVolumePaths) Name() string {
	return "InitializeVolumePaths"
}

func (m *initializeVolumePaths) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		pattern := dyn.NewPattern(dyn.Key("resources"), dyn.Key("volumes"), dyn.AnyKey())
		return dyn.MapByPattern(root, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// volume_path is computed and read-only. Reject a user-provided value
			// instead of silently overwriting it with the computed path.
			if existing, ok := v.Get("volume_path").AsString(); ok && existing != "" {
				return dyn.InvalidValue, fmt.Errorf("%s.volume_path is computed and read-only; remove it from the configuration", p.String())
			}

			var vol resources.Volume
			if err := convert.ToTyped(&vol, v); err != nil {
				return dyn.InvalidValue, err
			}

			// Resolve references for the purpose of computing the path only; the
			// original field values in v are left untouched.
			vol.CatalogName = resolveResourceReference(root, vol.CatalogName)
			vol.SchemaName = resolveResourceReference(root, vol.SchemaName)
			vol.Name = resolveResourceReference(root, vol.Name)

			path := vol.ComputeVolumePath()
			if path == "" {
				return v, nil
			}
			return dyn.Set(v, "volume_path", dyn.V(path))
		})
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// resolveResourceReference returns the concrete value of a pure ${resources....} reference
// by looking it up in root. Non-reference values and references that cannot be resolved (or
// resolve to another reference) are returned unchanged, so they still contain "${" and the
// caller will not compute a volume_path for them.
func resolveResourceReference(root dyn.Value, s string) string {
	p, ok := dynvar.PureReferenceToPath(s)
	if !ok || p[0].Key() != "resources" {
		return s
	}
	rv, err := dyn.GetByPath(root, p)
	if err != nil {
		return s
	}
	rs, ok := rv.AsString()
	if !ok {
		return s
	}
	return rs
}
