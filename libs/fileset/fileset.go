package fileset

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/databricks/cli/libs/vfs"
)

// FileSet facilitates fast recursive file listing of a path.
// It optionally takes into account ignore rules through the [Ignorer] interface.
type FileSet struct {
	// Root path of the fileset.
	root vfs.Path

	// Ignorer interface to check if a file or directory should be ignored.
	ignore Ignorer
}

// New returns a [FileSet] for the given root path.
func New(root vfs.Path) *FileSet {
	return &FileSet{
		root:   root,
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

// Return all tracked files for Repo
func (w *FileSet) All() ([]File, error) {
	return w.recursiveListFiles()
}

// Recursively traverses dir in a depth first manner and returns a list of all files
// that are being tracked in the FileSet (ie not being ignored for matching one of the
// patterns in w.ignore)
func (w *FileSet) recursiveListFiles() (fileList []File, err error) {
	err = fs.WalkDir(w.root, ".", func(name string, d fs.DirEntry, err error) error {
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
			ign, err := w.ignore.IgnoreDirectory(name)
			if err != nil {
				return fmt.Errorf("cannot check if %s should be ignored: %w", name, err)
			}
			if ign {
				return fs.SkipDir
			}
			return nil
		}

		ign, err := w.ignore.IgnoreFile(name)
		if err != nil {
			return fmt.Errorf("cannot check if %s should be ignored: %w", name, err)
		}
		if ign {
			return nil
		}

		fileList = append(fileList, NewFile(w.root, d, name))
		return nil
	})
	return
}
