package git

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/vfs"
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
func (v *View) Ignore(relPath string) (bool, error) {
	// Retain trailing slash for directory patterns.
	// Needs special handling because it is removed by path cleaning.
	trailingSlash := ""
	if strings.HasSuffix(relPath, "/") {
		trailingSlash = "/"
	}

	return v.repo.Ignore(path.Join(v.targetPath, relPath) + trailingSlash)
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

func NewView(worktreeRoot, root vfs.Path) (*View, error) {
	repo, err := NewRepository(worktreeRoot)
	if err != nil {
		return nil, err
	}

	// Target path must be relative to the repository root path.
	target := root.Native()
	prefix := repo.rootDir.Native()
	if !strings.HasPrefix(target, prefix) {
		return nil, fmt.Errorf("path %q is not within repository root %q", root.Native(), prefix)
	}

	// Make target a relative path.
	target = strings.TrimPrefix(target, prefix)
	target = strings.TrimPrefix(target, string(os.PathSeparator))
	target = path.Clean(filepath.ToSlash(target))

	result := &View{
		repo:       repo,
		targetPath: target,
	}

	result.SetupDefaults()
	return result, nil
}

func NewViewAtRoot(root vfs.Path) (*View, error) {
	return NewView(root, root)
}

func (v *View) SetupDefaults() {
	// Hard code .databricks ignore pattern so that we never sync it (irrespective)
	// of .gitignore patterns
	v.repo.addIgnoreRule(newStringIgnoreRules([]string{
		".databricks",
	}))

	v.repo.taintIgnoreRules()
}
