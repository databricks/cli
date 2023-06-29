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

	return fs.WalkDir(zipReader, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, path)
		if d.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		targetFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}

		sourceFile, err := zipReader.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(targetFile, sourceFile)
		return err
	})
}
