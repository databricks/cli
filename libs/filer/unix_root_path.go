package filer

import (
	"fmt"
	"path"
	"strings"
)

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
