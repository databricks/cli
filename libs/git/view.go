package git

import (
	"os"
	"path/filepath"
	"strings"
)

// View represents a view on a directory tree that takes into account
// all applicable .gitignore files. The directory tree does NOT need
// to be the repository root.
//
// For example: with a repository root at "myrepo", a view can be
// anchored at "myrepo/someproject" and still respect the ignore
// rules defined at "myrepo/.gitignore".
//
// We use this functionality to synchronize files from a path nested
// in a repository while respecting the repository's ignore rules.
type View struct {
	// repo points to the repository that contains the directory
	// that this view is anchored at.
	repo *Repository

	// targetPath is the relative path to the directory tree that this
	// view is anchored at (with respect to the repository root).
	// For example: "." or "a/b".
	targetPath string
}

// Ignore computes whether to ignore the specified path.
// The specified path is relative to the view's target path.
func (v *View) Ignore(path string) (bool, error) {
	path = filepath.ToSlash(path)

	// Retain trailing slash for directory patterns.
	// Needs special handling because it is removed by path cleaning.
	trailingSlash := ""
	if strings.HasSuffix(path, "/") {
		trailingSlash = "/"
	}

	return v.repo.Ignore(filepath.Join(v.targetPath, path) + trailingSlash)
}

// IgnoreFile returns if the gitignore rules in this fileset
// apply to the specified file path.
//
// This function is provided to implement [fileset.Ignorer].
func (v *View) IgnoreFile(file string) (bool, error) {
	return v.Ignore(file)
}

// IgnoreDirectory returns if the gitignore rules in this fileset
// apply to the specified directory path.
//
// A gitignore rule may apply only to directories if it uses
// a trailing slash. Therefore this function checks the gitignore
// rules for the specified directory path first without and then
// with a trailing slash.
//
// This function is provided to implement [fileset.Ignorer].
func (v *View) IgnoreDirectory(dir string) (bool, error) {
	ign, err := v.Ignore(dir)
	if err != nil {
		return false, err
	}
	if ign {
		return ign, nil
	}
	return v.Ignore(dir + "/")
}

func NewView(path string) (*View, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	repo, err := NewRepository(path)
	if err != nil {
		return nil, err
	}

	// Target path must be relative to the repository root path.
	targetPath, err := filepath.Rel(repo.rootPath, path)
	if err != nil {
		return nil, err
	}

	return &View{
		repo:       repo,
		targetPath: targetPath,
	}, nil
}

func (v *View) EnsureValidGitIgnoreExists() error {
	ign, err := v.IgnoreDirectory(".databricks")
	if err != nil {
		return err
	}

	// return early if .databricks is already being ignored
	if ign {
		return nil
	}

	// Create .gitignore with .databricks entry
	gitIgnorePath := filepath.Join(v.repo.Root(), v.targetPath, ".gitignore")
	file, err := os.OpenFile(gitIgnorePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Hard code .databricks ignore pattern so that we never sync it (irrespective)
	// of .gitignore patterns
	v.repo.addIgnoreRule(newStringIgnoreRules([]string{
		".databricks",
	}))

	_, err = file.WriteString("\n.databricks\n")
	if err != nil {
		return err
	}

	v.repo.taintIgnoreRules()
	return nil
}
