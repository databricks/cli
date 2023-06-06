package filer

import (
	"fmt"
	"path"
	"strings"
)

// RootPath can be joined with a relative path and ensures that
// the returned path is always a strict child of the root path.
type RootPath struct {
	rootPath string
}

// NewRootPath constructs and returns [RootPath].
// The named path is cleaned on construction.
func NewRootPath(name string) RootPath {
	return RootPath{
		rootPath: path.Clean(name),
	}
}

// Join returns the specified path name joined to the root.
// It returns an error if the resulting path is not a strict child of the root path.
func (p *RootPath) Join(name string) (string, error) {
	absPath := path.Join(p.rootPath, name)

	// Don't allow escaping the specified root using relative paths.
	if !strings.HasPrefix(absPath, p.rootPath) {
		return "", fmt.Errorf("relative path escapes root: %s", name)
	}

	return absPath, nil
}
