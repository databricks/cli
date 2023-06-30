package zip

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Extract(src string, dst string) error {
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	// create dst directory incase it does not  exist.
	err = os.MkdirAll(dst, 0755)
	if err != nil {
		return err
	}

	return fs.WalkDir(zipReader, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, path)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		targetFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer targetFile.Close()

		sourceFile, err := zipReader.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		_, err = io.Copy(targetFile, sourceFile)
		return err
	})
}
