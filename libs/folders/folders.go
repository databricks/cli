package folders

import (
	"errors"
	"os"
	"path/filepath"
)

// FindDirWithLeaf returns the first directory that holds `leaf`,
// traversing up to the root of the filesystem, starting at `dir`.
func FindDirWithLeaf(dir string, leaf string) (string, error) {
	for {
		_, err := os.Stat(filepath.Join(dir, leaf))

		// No error means we found the leaf in dir.
		if err == nil {
			return dir, nil
		}

		// ErrNotExist means we continue traversal up the tree.
		if errors.Is(err, os.ErrNotExist) {
			next := filepath.Dir(dir)
			if dir == next {
				// Return if we cannot continue traversal.
				return "", err
			}

			dir = next
			continue
		}

		return "", err
	}
}
