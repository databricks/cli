package testarchive

import (
	"fmt"
	"os"
	"path/filepath"
)

// Bundle acceptance tests pin UV_PYTHON=3.10 and run uv offline, but serverless only has Python 3.12.
const (
	pythonVersion = "3.10.16"
	pythonBuild   = "20250317"
)

// PythonDownloader downloads a python-build-standalone CPython into <BinDir>/<Arch>/python.
type PythonDownloader struct {
	BinDir string
	Arch   string
}

func (p PythonDownloader) Download() error {
	arch, err := UvDownloader{}.mapArchitecture(p.Arch)
	if err != nil {
		return err
	}

	dir := filepath.Join(p.BinDir, p.Arch)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://github.com/astral-sh/python-build-standalone/releases/download/%s/cpython-%s%%2B%s-%s-unknown-linux-gnu-install_only_stripped.tar.gz",
		pythonBuild, pythonVersion, pythonBuild, arch,
	)
	tempFile := filepath.Join(dir, "python.tar.gz")
	if err := downloadFile(url, tempFile); err != nil {
		return err
	}

	// The archive has a single top-level "python" directory.
	if err := ExtractTarGz(tempFile, dir); err != nil {
		return err
	}
	return os.Remove(tempFile)
}
