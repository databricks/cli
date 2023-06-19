package filer

import (
	"fmt"
	"path/filepath"
	"strings"
)

type WindowsRootPath struct {
	rootPath string
}

func NewWindowsRootPath(name string) WindowsRootPath {
	// Windows file systems do not have a "root" directory. Instead paths require
	// a Volume/Drive letter specified. If a user of this struct specifies "/" then
	// we treat it as the "root" and skip any validation
	if name == "/" {
		return WindowsRootPath{""}
	}

	return WindowsRootPath{filepath.Clean(name)}
}

// Join returns the specified path name joined to the root.
// It returns an error if the resulting path is not a strict child of the root path.
func (p WindowsRootPath) Join(name string) (string, error) {
	absPath := filepath.Join(p.rootPath, name)

	// Don't allow escaping the specified root using relative paths.
	if !strings.HasPrefix(absPath, p.rootPath) {
		return "", fmt.Errorf("relative path %s escapes root %s", name, p.rootPath)
	}

	return absPath, nil
}

func (p WindowsRootPath) Root() string {
	return p.rootPath
}
