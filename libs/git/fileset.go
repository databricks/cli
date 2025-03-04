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

// NewFileSet returns [FileSet] for the directory `root` which is contained within Git worktree located at `worktreeRoot`.
func NewFileSet(worktreeRoot, root vfs.Path, paths ...[]string) (*FileSet, error) {
	fs := fileset.New(root, paths...)
	v, err := NewView(worktreeRoot, root)
	if err != nil {
		return nil, err
	}
	fs.SetIgnorer(v)
	return &FileSet{
		fileset: fs,
		view:    v,
	}, nil
}

func NewFileSetAtRoot(root vfs.Path, paths ...[]string) (*FileSet, error) {
	return NewFileSet(root, root, paths...)
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
