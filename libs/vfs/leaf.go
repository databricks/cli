package vfs

import (
	"errors"
	"io/fs"
)

// FindLeafInTree returns the first path that holds `name`,
// traversing up to the root of the filesystem, starting at `p`.
func FindLeafInTree(p Path, name string) (Path, error) {
	for p != nil {
		_, err := fs.Stat(p, name)

		// No error means we found the leaf in p.
		if err == nil {
			return p, nil
		}

		// ErrNotExist means we continue traversal up the tree.
		if errors.Is(err, fs.ErrNotExist) {
			p = p.Parent()
			continue
		}

		return nil, err
	}

	return nil, fs.ErrNotExist
}
