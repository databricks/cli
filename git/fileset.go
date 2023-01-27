package git

import (
	"os"
	"path/filepath"

	"github.com/databricks/bricks/libs/fileset"
)

// Retuns FileSet for the git repo located at `root`
func NewFileSet(root string) (*fileset.FileSet, error) {
	w := fileset.New(root)
	v, err := NewView(root)
	if err != nil {
		return nil, err
	}
	w.SetIgnorer(v)
	return w, nil
}

// Only call this function for a bricks project root
// since it will create a .gitignore file if missing
func EnsureValidGitIgnoreExists(w *fileset.FileSet) error {
	if w.Ignorer().IgnoreDirectory(".databricks") {
		return nil
	}

	gitIgnorePath := filepath.Join(w.Root(), ".gitignore")
	f, err := os.OpenFile(gitIgnorePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n.databricks\n")
	if err != nil {
		return err
	}

	// Reload view to update ignore rules.
	v, err := NewView(w.Root())
	if err != nil {
		return err
	}
	w.SetIgnorer(v)
	return nil
}
