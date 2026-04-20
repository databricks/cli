package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type ConfigFileNames []string

// FileNames contains allowed names of the root ucm configuration file.
var FileNames = ConfigFileNames{
	"ucm.yml",
	"ucm.yaml",
}

func (c ConfigFileNames) FindInPath(path string) (string, error) {
	result := ""
	var firstErr error

	for _, file := range c {
		filePath := filepath.Join(path, file)
		_, err := os.Stat(filePath)
		if err == nil {
			if result != "" {
				return "", fmt.Errorf("multiple ucm root configuration files found in %s", path)
			}
			result = filePath
		} else if firstErr == nil {
			firstErr = err
		}
	}

	if result == "" {
		return "", firstErr
	}

	return result, nil
}
