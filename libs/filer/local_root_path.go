package filer

import (
	"fmt"
	"path/filepath"
	"strings"
)

type localRootPath struct {
	rootPath string
}

func NewLocalRootPath(root string) localRootPath {
	if root == "" {
		return localRootPath{""}
	}
	return localRootPath{filepath.Clean(root)}
}

func (rp *localRootPath) Join(name string) (string, error) {
	absPath := filepath.Join(rp.rootPath, name)
	if !strings.HasPrefix(absPath, rp.rootPath) {
		return "", fmt.Errorf("relative path escapes root: %s", name)
	}
	return absPath, nil
}
