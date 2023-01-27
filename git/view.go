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
	repo *Repository

	// targetPath is the relative path within the repository we care about.
	// For example: "." or "a/b".
	targetPath string
}

// Ignore computes whether to ignore the specified path.
// The specified path is relative to the view's target path.
func (v *View) Ignore(path string) bool {
	// Retain trailing slash for directory patterns.
	// Needs special handling because it is removed by path cleaning.
	trailingSlash := ""
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		trailingSlash = string(os.PathSeparator)
	}

	return v.repo.Ignore(filepath.Join(v.targetPath, path) + trailingSlash)
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

	// Load ignore files relevant for this view's path.
	err = repo.includeIgnoreFilesForPath(targetPath)
	if err != nil {
		return nil, err
	}

	view := &View{
		repo:       repo,
		targetPath: targetPath,
	}

	return view, nil
}
