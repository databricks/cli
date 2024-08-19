package fileset

import (
	"fmt"
	"io/fs"
	pathlib "path"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/libs/vfs"
)

// FileSet facilitates recursive file listing for paths rooted at a given directory.
// It optionally takes into account ignore rules through the [Ignorer] interface.
type FileSet struct {
	// Root path of the fileset.
	root vfs.Path

	// Paths to include in the fileset.
	// Files are included as-is (if not ignored) and directories are traversed recursively.
	// Defaults to []string{"."} if not specified.
	paths []string

	// Ignorer interface to check if a file or directory should be ignored.
	ignore Ignorer
}

// New returns a [FileSet] for the given root path.
// It optionally accepts a list of paths relative to the root to include in the fileset.
// If not specified, it defaults to including all files in the root path.
func New(root vfs.Path, args ...[]string) *FileSet {
	// Default to including all files in the root path.
	if len(args) == 0 {
		args = [][]string{{"."}}
	}

	// Collect list of normalized and cleaned paths.
	var paths []string
	for _, arg := range args {
		for _, path := range arg {
			path = filepath.ToSlash(path)
			path = pathlib.Clean(path)

			// Skip path if it's already in the list.
			if slices.Contains(paths, path) {
				continue
			}

			paths = append(paths, path)
		}
	}

	return &FileSet{
		root:   root,
		paths:  paths,
		ignore: nopIgnorer{},
	}
}

// Ignorer returns the [FileSet]'s current ignorer.
func (w *FileSet) Ignorer() Ignorer {
	return w.ignore
}

// SetIgnorer sets the [Ignorer] interface for this [FileSet].
func (w *FileSet) SetIgnorer(ignore Ignorer) {
	w.ignore = ignore
}

// Files returns performs recursive listing on all configured paths and returns
// the collection of files it finds (and are not ignored).
// The returned slice does not contain duplicates.
// The order of files in the slice is stable.
func (w *FileSet) Files() (out []File, err error) {
	seen := make(map[string]struct{})
	for _, p := range w.paths {
		files, err := w.recursiveListFiles(p, seen)
		if err != nil {
			return nil, err
		}
		out = append(out, files...)
	}
	return out, nil
}

// Recursively traverses dir in a depth first manner and returns a list of all files
// that are being tracked in the FileSet (ie not being ignored for matching one of the
// patterns in w.ignore)
func (w *FileSet) recursiveListFiles(path string, seen map[string]struct{}) (out []File, err error) {
	err = fs.WalkDir(w.root, path, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		switch {
		case info.Mode().IsDir():
			ign, err := w.ignore.IgnoreDirectory(name)
			if err != nil {
				return fmt.Errorf("cannot check if %s should be ignored: %w", name, err)
			}
			if ign {
				return fs.SkipDir
			}

		case info.Mode().IsRegular():
			ign, err := w.ignore.IgnoreFile(name)
			if err != nil {
				return fmt.Errorf("cannot check if %s should be ignored: %w", name, err)
			}
			if ign {
				return nil
			}

			// Skip duplicates
			if _, ok := seen[name]; ok {
				return nil
			}

			seen[name] = struct{}{}
			out = append(out, NewFile(w.root, d, name))

		default:
			// Skip non-regular files (e.g. symlinks).
		}

		return nil
	})
	return
}
