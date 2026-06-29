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

// InitializeVolumePaths sets resources.volumes.*.volume_path from catalog_name, schema_name, and name.
//
// References in those fields are resolved locally only to compute the path; the original field
// values are left intact so validate and plan still show the references. A component that cannot be
// resolved locally is embedded verbatim as a ${...} reference and resolved later during plan or
// deploy, like any other resource reference.
//
// Must run exactly once: volume_path is computed here and never persisted, so a second run would
// see its own value and trip the "computed and read-only" rejection below.
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
			// volume_path is computed and read-only; reject a user-provided value instead of overwriting it.
			if existing, ok := v.Get("volume_path").AsString(); ok && existing != "" {
				return dyn.InvalidValue, fmt.Errorf("%s.volume_path is computed and read-only; remove it from the configuration", p.String())
			}

			var vol resources.Volume
			if err := convert.ToTyped(&vol, v); err != nil {
				return dyn.InvalidValue, err
			}

			// Resolve references to compute the path only; the field values in v are left untouched.
			vol.CatalogName = resolveResourceReference(root, vol.CatalogName)
			vol.SchemaName = resolveResourceReference(root, vol.SchemaName)
			vol.Name = resolveResourceReference(root, vol.Name)

			return dyn.Set(v, "volume_path", dyn.V(vol.ComputeVolumePath()))
		})
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// resolveResourceReference resolves a pure ${resources....} reference by looking it up in root.
// Values that are not such a reference, or cannot be resolved, are returned unchanged (still
// containing "${"), so the caller embeds the reference verbatim to be resolved later.
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
