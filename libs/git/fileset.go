package git

import (
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
)

// FileSet is Git repository aware implementation of [fileset.FileSet].
// It forces checking if gitignore files have been modified every
// time a call to [FileSet.Files] is made.
type FileSet struct {
	fileset *fileset.FileSet
	view    *View
}

// NewFileSet returns [FileSet] for the Git repository located at `root`.
func NewFileSet(root vfs.Path, paths ...[]string) (*FileSet, error) {
	fs := fileset.New(root, paths...)
	v, err := NewView(root)
	if err != nil {
		return nil, err
	}
	fs.SetIgnorer(v)
	return &FileSet{
		fileset: fs,
		view:    v,
	}, nil
}

func (f *FileSet) IgnoreFile(file string) (bool, error) {
	return f.view.IgnoreFile(file)
}

func (f *FileSet) IgnoreDirectory(dir string) (bool, error) {
	return f.view.IgnoreDirectory(dir)
}

func (f *FileSet) Files() ([]fileset.File, error) {
	f.view.repo.taintIgnoreRules()
	return f.fileset.Files()
}

func (f *FileSet) EnsureValidGitIgnoreExists() error {
	return f.view.EnsureValidGitIgnoreExists()
}
