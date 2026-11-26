package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type ConfigFileNames []string

// FileNames contains the names used for automatic root configuration discovery.
var FileNames = ConfigFileNames{
	"databricks.yml",
	"databricks.yaml",
	"bundle.yml",
	"bundle.yaml",
}

// IsConfigFile reports whether path has a supported bundle configuration extension.
func IsConfigFile(path string) bool {
	switch filepath.Ext(path) {
	case ".yml", ".yaml", ".json":
		return true
	}
	return false
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
