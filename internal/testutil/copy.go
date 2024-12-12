package testutil

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/require"
)

// CopyDirectory copies the contents of a directory to another directory.
// The destination directory is created if it does not exist.
func CopyDirectory(t TestingT, src, dst string) {
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		require.NoError(t, err)

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}

		// Copy the file to the temporary directory
		in, err := os.Open(path)
		if err != nil {
			return err
		}

		defer in.Close()

		out, err := os.Create(filepath.Join(dst, rel))
		if err != nil {
			return err
		}

		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})

	require.NoError(t, err)
}
