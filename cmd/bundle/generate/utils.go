package generate

import (
	"path/filepath"
)

// makeRelativeToRoot converts a path to be relative to the bundle root.
// If the path is already relative, it is returned as-is.
// If the path is absolute and under the root, it is made relative.
// This is needed because the output filer is rooted at the bundle root,
// and paths must be relative to that root for the filer to write correctly.
func makeRelativeToRoot(root, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return path, nil
	}

	return filepath.Rel(root, path)
}
