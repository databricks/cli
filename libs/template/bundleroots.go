package template

import (
	"path"
	"slices"
)

// bundleConfigFileNames are the file names that mark a directory as a bundle
// root. This mirrors bundle/config.FileNames, which is the source of truth. It
// is duplicated here because libs/ does not depend on bundle/; keep the two in
// sync when a new root configuration file name is introduced.
var bundleConfigFileNames = []string{
	"databricks.yml",
	"databricks.yaml",
	"bundle.yml",
	"bundle.yaml",
}

// bundleRoots returns the directories that received a bundle configuration file,
// derived from the slash-separated, output-relative paths of the files that were
// actually persisted. The result is de-duplicated and sorted, and a bundle root
// that is the output directory itself is reported as ".".
//
// The paths are treated as slash-separated (not OS-specific) because they are
// relative to the output filer, which may be either the local file system or the
// workspace file system.
func bundleRoots(persistedPaths []string) []string {
	var roots []string

	for _, p := range persistedPaths {
		if !slices.Contains(bundleConfigFileNames, path.Base(p)) {
			continue
		}

		dir := path.Dir(p)
		if dir == "" {
			dir = "."
		}
		if !slices.Contains(roots, dir) {
			roots = append(roots, dir)
		}
	}

	slices.Sort(roots)
	return roots
}
