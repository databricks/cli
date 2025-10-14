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
	root    vfs.Path
	paths   []string
}

// NewFileSet returns [FileSet] for the directory `root` which is contained within Git worktree located at `worktreeRoot`.
func NewFileSet(worktreeRoot, root vfs.Path, paths ...[]string) (*FileSet, error) {
	fs := fileset.New(root, paths...)
	v, err := NewView(worktreeRoot, root)
	if err != nil {
		return nil, err
	}
	fs.SetIgnorer(v)

	var p []string
	if len(paths) > 0 {
		p = paths[0]
	}

	return &FileSet{
		fileset: fs,
		view:    v,
		root:    root,
		paths:   p,
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

// FilesWithStats returns files along with statistics about what was skipped during the walk.
func (f *FileSet) FilesWithStats() ([]fileset.File, fileset.WalkStats, error) {
	f.view.repo.taintIgnoreRules()
	return f.fileset.FilesWithStats()
}

// AllFiles returns all files in the fileset without applying gitignore rules.
func (f *FileSet) AllFiles() ([]fileset.File, error) {
	// Create a new fileset without the gitignore ignorer to get all files
	// If paths is empty, pass no arguments to get the default behavior
	var fs *fileset.FileSet
	if len(f.paths) == 0 {
		fs = fileset.New(f.root)
	} else {
		fs = fileset.New(f.root, f.paths)
	}
	return fs.Files()
}

// TaintIgnoreRules marks gitignore rules as needing reload on next use.
func (f *FileSet) TaintIgnoreRules() {
	f.view.repo.taintIgnoreRules()
}
