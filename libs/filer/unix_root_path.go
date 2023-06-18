package filer

import (
	"fmt"
	"path"
	"strings"
)

// UnixRootPath can be joined with a relative path and ensures that
// the returned path is always a strict child of the root path.
type UnixRootPath struct {
	rootPath string
}

// NewUnixRootPath constructs and returns [UnixRootPath].
// The named path is cleaned on construction.
func NewUnixRootPath(name string) UnixRootPath {
	return UnixRootPath{
		rootPath: path.Clean(name),
	}
}

// Join returns the specified path name joined to the root.
// It returns an error if the resulting path is not a strict child of the root path.
func (p UnixRootPath) Join(name string) (string, error) {
	absPath := path.Join(p.rootPath, name)

	// Don't allow escaping the specified root using relative paths.
	if !strings.HasPrefix(absPath, p.rootPath) {
		return "", fmt.Errorf("relative path escapes root: %s", name)
	}

	return absPath, nil
}

func (p UnixRootPath) Root() string {
	return p.rootPath
}
