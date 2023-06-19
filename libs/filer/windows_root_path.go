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
	return WindowsRootPath{filepath.Clean(name)}
}

func (p WindowsRootPath) Join(name string) (string, error) {
	// Windows file systems do not have a "root" directory. Instead paths require
	// a Volume/Drive letter specified. If a user of this struct specifies "/" then
	// we treat it as the "root" and skip any validation
	if p.rootPath == "/" {
		return name, nil
	}

	absPath := filepath.Join(p.rootPath, name)

	// Don't allow escaping the specified root using relative paths.
	if !strings.HasPrefix(absPath, p.rootPath) {
		return "", fmt.Errorf("relative path escapes root: %s", name)
	}

	return absPath, nil
}

func (p WindowsRootPath) Root() string {
	return p.rootPath
}
