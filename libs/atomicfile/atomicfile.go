// Package atomicfile writes a file so that readers and crashes never observe a
// partial result: the data goes to a temporary file in the same directory and
// is then renamed over the destination. The rename is atomic on a single
// filesystem, which is why the temp file must live next to the target rather
// than in os.TempDir (a cross-filesystem rename fails with EXDEV).
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write atomically writes data to path with mode perm. It creates a temp file
// in path's directory, writes and chmods it, then renames it over path.
//
// The resulting file always has mode perm; it does not inherit the permissions
// of a file it replaces (os.CreateTemp starts at 0600, and the rename replaces
// the inode, so callers state the mode they want explicitly). The parent
// directory must already exist.
func Write(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Temp file in the same directory so the rename stays on one filesystem.
	// The "." prefix hides a leftover temp file and the ".tmp" suffix lets
	// callers sweep leftovers with a "*.tmp" glob.
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("chmod %s: %w", tmpPath, err)
	}
	// Close before the rename: on Windows the file must not be open.
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}
	return nil
}
