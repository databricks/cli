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

// Option configures a Write call.
type Option func(*config)

type config struct {
	mkdirPerm os.FileMode
	mkdir     bool
}

// MkDir makes Write create path's parent directory (and any missing parents)
// with mode perm before writing, like os.MkdirAll. The directory mode is a
// separate, deliberate choice from the file mode (e.g. 0o700 for a directory
// holding secrets), so callers pass it explicitly rather than it defaulting.
func MkDir(perm os.FileMode) Option {
	return func(c *config) {
		c.mkdir = true
		c.mkdirPerm = perm
	}
}

// Write atomically writes data to path with mode perm. It creates a temp file
// in path's directory, writes and chmods it, then renames it over path.
//
// The resulting file always has mode perm; it does not inherit the permissions
// of a file it replaces (os.CreateTemp starts at 0600, and the rename replaces
// the inode, so callers state the mode they want explicitly). The parent
// directory must already exist unless the MkDir option is passed.
func Write(path string, data []byte, perm os.FileMode, opts ...Option) error {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	dir := filepath.Dir(path)

	if cfg.mkdir {
		if err := os.MkdirAll(dir, cfg.mkdirPerm); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

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
