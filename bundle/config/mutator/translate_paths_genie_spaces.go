package mutator

import (
	"context"

	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/dyn"
)

func (t *translateContext) applyGenieSpaceTranslations(ctx context.Context, v dyn.Value) (dyn.Value, error) {
	// Rewrite the `file_path` field to a path relative to the bundle sync root.
	// We load the file at this path and use its contents for the genie space contents.

	return paths.VisitGenieSpacePaths(v, func(p dyn.Path, mode paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
		opts := translateOptions{
			Mode: mode,
		}

		return t.rewriteValue(ctx, p, v, t.b.BundleRootPath, opts)
	})
}
