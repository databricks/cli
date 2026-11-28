package skills

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

// dirEnv names a folder that holds a skills release. It is read before any
// other source, so a developer can point the CLI at a local copy.
const dirEnv = "DATABRICKS_SKILLS_DIR"

// ErrNoSource reports that this build has no skills source to read from.
var ErrNoSource = errors.New("no skills source is available in this build")

// Resolve returns the source that serves skills. For now that is the folder in
// DATABRICKS_SKILLS_DIR, when it is set. Otherwise no source is available.
func Resolve(ctx context.Context) (Source, error) {
	dir := env.Get(ctx, dirEnv)
	if dir == "" {
		return nil, ErrNoSource
	}
	return NewFSSource(os.DirFS(dir), filepath.ToSlash(dir)), nil
}

// LocalCatalog returns the freshest catalog that needs no network call. For now
// that is the folder in DATABRICKS_SKILLS_DIR. It returns nil when no folder is
// set.
func LocalCatalog(ctx context.Context) (*Catalog, error) {
	src, err := Resolve(ctx)
	if errors.Is(err, ErrNoSource) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return src.Catalog(ctx)
}
