package filer

import (
	"fmt"
	"path"
	"strings"
)

// WorkspaceRootPath can be joined with a relative path and ensures that
// the returned path is always a strict child of the root path.
type WorkspaceRootPath struct {
	rootPath string
}

// NewWorkspaceRootPath constructs and returns [RootPath].
// The named path is cleaned on construction.
func NewWorkspaceRootPath(name string) WorkspaceRootPath {
	return WorkspaceRootPath{
		rootPath: path.Clean(name),
	}
}

// Join returns the specified path name joined to the root.
// It returns an error if the resulting path is not a strict child of the root path.
func (p *WorkspaceRootPath) Join(name string) (string, error) {
	absPath := path.Join(p.rootPath, name)

	// Don't allow escaping the specified root using relative paths.
	// Joining exactly the root must stay allowed: calls like ReadDir(".") resolve to it.
	// Any other path must extend the root past a separator boundary; a plain prefix
	// check would also accept siblings like "/root-evil" for root "/root".
	// The suffix guard covers filers rooted at "/" (see cmd/fs), where the cleaned
	// root already ends in a separator.
	root := p.rootPath
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	if absPath != p.rootPath && !strings.HasPrefix(absPath, root) {
		return "", fmt.Errorf("relative path escapes root: %s", name)
	}

	return absPath, nil
}
