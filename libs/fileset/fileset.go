package fileset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileSet facilitates fast recursive file listing of a path.
// It optionally takes into account ignore rules through the [Ignorer] interface.
type FileSet struct {
	root   string
	ignore Ignorer
}

// New returns a [FileSet] for the given root path.
func New(root string) *FileSet {
	return &FileSet{
		root:   filepath.Clean(root),
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

// Return root for fileset.
func (w *FileSet) Root() string {
	return w.root
}

// Return all tracked files for Repo
func (w *FileSet) All() ([]File, error) {
	return w.RecursiveListFiles(w.root)
}

// Recursively traverses dir in a depth first manner and returns a list of all files
// that are being tracked in the FileSet (ie not being ignored for matching one of the
// patterns in w.ignore)
func (w *FileSet) RecursiveListFiles(dir string) (fileList []File, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(w.root, path)
		if err != nil {
			return err
		}

		// skip symlinks
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			ign, err := w.ignore.IgnoreDirectory(relPath)
			if err != nil {
				return fmt.Errorf("cannot check if %s should be ignored: %w", relPath, err)
			}
			if ign {
				return filepath.SkipDir
			}
			return nil
		}

		ign, err := w.ignore.IgnoreFile(relPath)
		if err != nil {
			return fmt.Errorf("cannot check if %s should be ignored: %w", relPath, err)
		}
		if ign {
			return nil
		}

		fileList = append(fileList, File{d, path, relPath})
		return nil
	})
	return
}
