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

		CopyFile(t, path, filepath.Join(dst, rel))
		return nil
	})

	require.NoError(t, err)
}

func CopyFile(t TestingT, src, dst string) {
	srcF, err := os.Open(src)
	require.NoError(t, err)
	defer srcF.Close()

	dstF, err := os.Create(dst)
	require.NoError(t, err)
	defer dstF.Close()

	_, err = io.Copy(dstF, srcF)
	require.NoError(t, err)
}
