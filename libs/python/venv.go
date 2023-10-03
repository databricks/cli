package python

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrNoVirtualEnvDetected = errors.New("no Python virtual environment detected")

// DetectVirtualEnv scans direct subfolders in path to get a valid
// Virtual Environment installation, that is marked by pyvenv.cfg file.
//
// See: https://packaging.python.org/en/latest/tutorials/packaging-projects/
func DetectVirtualEnvPath(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	for _, v := range files {
		if !v.IsDir() {
			continue
		}
		candidate := filepath.Join(path, v.Name())
		_, err = os.Stat(filepath.Join(candidate, "pyvenv.cfg"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return candidate, nil
	}
	return "", ErrNoVirtualEnvDetected
}
