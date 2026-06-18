package snapshot

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type translateResourcePaths struct{}

// TranslateResourcePaths replaces local absolute paths in resource configs with the
// remote snapshot path. It must run after snapshot.Upload() has set
// b.Config.Workspace.FilePath to the content-addressed snapshot location.
//
// translate_paths.go uses b.SyncRootPath as the remote root for immutable bundles,
// so resource paths are stored as local absolute paths until this mutator rewrites them.
func TranslateResourcePaths() bundle.Mutator {
	return &translateResourcePaths{}
}

func (m *translateResourcePaths) Name() string { return "snapshot.TranslateResourcePaths" }

func (m *translateResourcePaths) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	localPrefix := b.SyncRootPath + "/"
	remotePrefix := b.Config.Workspace.FilePath + "/"

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if len(p) == 0 {
				return v, nil
			}
			// Only rewrite paths inside the resources section.
			if p[0] != dyn.Key("resources") {
				return v, dyn.ErrSkip
			}
			str, ok := v.AsString()
			if !ok {
				return v, nil
			}
			if !strings.HasPrefix(str, localPrefix) {
				return v, nil
			}
			return dyn.NewValue(remotePrefix+strings.TrimPrefix(str, localPrefix), v.Locations()), nil
		})
	})
	return diag.FromErr(err)
}
