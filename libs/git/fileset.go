package git

import (
	"github.com/databricks/bricks/libs/fileset"
)

// FileSet is Git repository aware implementation of [fileset.FileSet].
// It forces checking if gitignore files have been modified every
// time a call to [FileSet.All] or [FileSet.RecursiveListFiles] is made.
type FileSet struct {
	fileset *fileset.FileSet
	view    *View
}

// NewFileSet returns [FileSet] for the Git repository located at `root`.
func NewFileSet(root string) (*FileSet, error) {
	fs := fileset.New(root)
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

func (f *FileSet) Root() string {
	return f.fileset.Root()
}

func (f *FileSet) All() ([]fileset.File, error) {
	f.view.repo.taintIgnoreRules()
	return f.fileset.All()
}

func (f *FileSet) RecursiveListFiles(dir string) ([]fileset.File, error) {
	f.view.repo.taintIgnoreRules()
	return f.fileset.RecursiveListFiles(dir)
}

func (f *FileSet) EnsureValidGitIgnoreExists() error {
	return f.view.EnsureValidGitIgnoreExists()
}
