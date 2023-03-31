package git

import (
	"fmt"
	"os"
	"path/filepath"

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
	ign, err := f.view.IgnoreDirectory(".databricks")
	if err != nil {
		return nil, err
	}
	if !ign {
		return nil, fmt.Errorf("cannot sync because .databricks is not present in .gitignore")
	}
	return f.fileset.All()
}

func (f *FileSet) RecursiveListFiles(dir string) ([]fileset.File, error) {
	f.view.repo.taintIgnoreRules()
	return f.fileset.RecursiveListFiles(dir)
}

// Only call this function for a bricks project root
// since it will create a .gitignore file if missing
func (f *FileSet) EnsureValidGitIgnoreExists() error {
	ign, err := f.view.IgnoreDirectory(".databricks")
	if err != nil {
		return err
	}
	if ign {
		return nil
	}

	gitIgnorePath := filepath.Join(f.fileset.Root(), ".gitignore")
	file, err := os.OpenFile(gitIgnorePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString("\n.databricks\n")
	if err != nil {
		return err
	}

	f.view.repo.taintIgnoreRules()
	return nil
}
