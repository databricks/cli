package filer

import (
	"fmt"
	"path"
	"strings"
)

// RootPath can be joined with a relative path and ensures that
// the returned path is always a strict child of the root path.
type RootPath string

// Join returns the specified path name joined to the root.
// It returns an error if the resulting path is not a strict child of the root path.
func (p RootPath) Join(name string) (string, error) {
	rootPath := path.Clean(string(p))
	absPath := path.Join(rootPath, name)

	// Don't allow escaping the specified root using relative paths.
	if !strings.HasPrefix(absPath, rootPath) {
		return "", fmt.Errorf("relative path escapes root: %s", name)
	}

	// Don't allow name to resolve to the root path.
	if strings.TrimPrefix(absPath, rootPath) == "" {
		return "", fmt.Errorf("relative path resolves to root: %s", name)
	}

	return absPath, nil
}
