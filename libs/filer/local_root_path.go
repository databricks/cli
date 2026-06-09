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

	// An empty root carries no restriction (cmd/fs constructs unrooted local filers).
	if rp.rootPath == "" {
		return absPath, nil
	}

	// Don't allow escaping the specified root using relative paths.
	// Joining exactly the root must stay allowed: calls like ReadDir(".") resolve to it.
	// Any other path must extend the root past a separator boundary; a plain prefix
	// check would also accept siblings like "/root-evil" for root "/root".
	// The suffix guard covers roots that already end in a separator after
	// cleaning ("/" or a Windows drive root like `C:\`).
	root := rp.rootPath
	if !strings.HasSuffix(root, string(filepath.Separator)) {
		root += string(filepath.Separator)
	}
	if absPath != rp.rootPath && !strings.HasPrefix(absPath, root) {
		return "", fmt.Errorf("relative path escapes root: %s", name)
	}
	return absPath, nil
}
