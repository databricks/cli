package git

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	fs.DirEntry
	Absolute, Relative string
}

func (f File) Modified() (ts time.Time) {
	info, err := f.Info()
	if err != nil {
		// return default time, beginning of epoch
		return ts
	}
	return info.ModTime()
}

// FileSet facilitates fast recursive tracked file listing
// with respect to patterns defined in `.gitignore` file
//
// root:   Root of the git repository
// ignore: List of patterns defined in `.gitignore`.
//
//	We do not sync files that match this pattern
type FileSet struct {
	root string
	view *View
}

// Retuns FileSet for the git repo located at `root`
func NewFileSet(root string) *FileSet {
	w := &FileSet{
		root: root,
	}
	err := w.createView()
	if err != nil {
		panic(err)
	}
	return w
}

// createView instantiates the view for this FileSet.
// Separate function because it needs to be recreated
// after adding a .gitignore entry in the below function.
func (w *FileSet) createView() error {
	view, err := NewView(w.root)
	if err != nil {
		return err
	}
	w.view = view
	return nil
}

// Only call this function for a bricks project root
// since it will create a .gitignore file if missing
func (w *FileSet) EnsureValidGitIgnoreExists() error {
	if w.view.Ignore(".databricks") {
		return nil
	}

	gitIgnorePath := filepath.Join(w.root, ".gitignore")
	f, err := os.OpenFile(gitIgnorePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n.databricks\n")
	if err != nil {
		return err
	}
	return w.createView()
}

// Return root for fileset.
func (w *FileSet) Root() string {
	return w.root
}

// Return all tracked files for Repo
func (w *FileSet) All() ([]File, error) {
	return w.RecursiveListFiles(w.root)
}

func (w *FileSet) IsGitIgnored(pattern string) bool {
	return w.view.Ignore(pattern)
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

		if w.view.Ignore(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			fileList = append(fileList, File{d, path, relPath})
		}

		return nil
	})
	return
}
