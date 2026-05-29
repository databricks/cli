package unpack

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const ownerRWXworldRX = 0o755

type GitHubZipball struct {
	io.Reader
}

func (v GitHubZipball) UnpackTo(libTarget string) error {
	raw, err := io.ReadAll(v)
	if err != nil {
		return err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return fmt.Errorf("zip: %w", err)
	}
	if len(zipReader.File) == 0 {
		return errors.New("empty zip archive")
	}
	// GitHub packages entire repo contents into a top-level folder, e.g. databrickslabs-ucx-2800c6b
	rootDirInZIP := zipReader.File[0].Name
	for _, zf := range zipReader.File {
		if zf.Name == rootDirInZIP {
			continue
		}
		normalizedName := strings.TrimPrefix(zf.Name, rootDirInZIP)
		if filepath.IsAbs(normalizedName) || strings.Contains(normalizedName, `\`) {
			return fmt.Errorf("invalid zip entry name: %q", zf.Name)
		}
		targetName := filepath.Join(libTarget, normalizedName)
		rel, err := filepath.Rel(libTarget, targetName)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("zip entry escapes target directory: %q", zf.Name)
		}
		if zf.FileInfo().IsDir() {
			err = os.MkdirAll(targetName, ownerRWXworldRX)
			if err != nil {
				return fmt.Errorf("mkdir %s: %w", normalizedName, err)
			}
			continue
		}
		err = v.extractFile(zf, targetName)
		if err != nil {
			return fmt.Errorf("extract %s: %w", zf.Name, err)
		}
	}
	return nil
}

func (v GitHubZipball) extractFile(zf *zip.File, targetName string) error {
	reader, err := zf.Open()
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	defer reader.Close()
	writer, err := os.OpenFile(targetName, os.O_CREATE|os.O_WRONLY, zf.Mode()&0o755)
	if err != nil {
		return fmt.Errorf("target: %w", err)
	}
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	return err
}
