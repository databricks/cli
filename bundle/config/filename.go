package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type ConfigFileNames []string

// FileNames contains allowed names of root bundle configuration files.
var FileNames = ConfigFileNames{
	"databricks.yml",
	"databricks.yaml",
	"bundle.yml",
	"bundle.yaml",
}

func (c ConfigFileNames) FindInPath(path string) (string, error) {
	result := ""

	for _, file := range c {
		filePath := filepath.Join(path, file)
		_, err := os.Stat(filePath)
		if err == nil {
			if result != "" {
				return "", fmt.Errorf("multiple bundle root configuration files found in %s", path)
			}
			result = filePath
		}
	}

	if result == "" {
		return "", fmt.Errorf("%s not found", c[0])
	}

	return result, nil
}
