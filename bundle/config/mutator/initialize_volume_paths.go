package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
	// Skip converting the configuration if there are no volumes.
	if len(b.Config.Resources.Volumes) == 0 {
		return nil
	}

	view := b.Config.View()
	pattern := structpath.MustParsePattern("resources.volumes.*")
	err := structvar.ForEach(view, pattern, func(p *structpath.PathNode, v structvar.View) error {
		// volume_path is computed and read-only; reject a user-provided value instead of overwriting it.
		if existing, ok := v.Get("volume_path").AsString(); ok && existing != "" {
			return fmt.Errorf("%s.volume_path is computed and read-only; remove it from the configuration", p.String())
		}

		// Resolve references to compute the path only; the field values are left untouched.
		vol := *b.Config.Resources.Volumes[p.KeyAt(2)]
		vol.CatalogName = resolveResourceReference(view, vol.CatalogName)
		vol.SchemaName = resolveResourceReference(view, vol.SchemaName)
		vol.Name = resolveResourceReference(view, vol.Name)

		return b.Config.Set(structpath.NewStringKey(p, "volume_path"), vol.ComputeVolumePath())
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// resolveResourceReference resolves a pure ${resources....} reference by looking it up in root.
// Values that are not such a reference, or cannot be resolved, are returned unchanged (still
// containing "${"), so the caller embeds the reference verbatim to be resolved later.
func resolveResourceReference(root structvar.View, s string) string {
	p, ok := structvar.PureReferenceToPath(s)
	if !ok || p.KeyAt(0) != "resources" {
		return s
	}
	rs, ok := root.Lookup(p).AsString()
	if !ok {
		return s
	}
	return rs
}
