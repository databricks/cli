package paths

import (
	"fmt"
	"path/filepath"
)

type Paths struct {
	// Absolute path on the local file system to the configuration file that holds
	// the definition of this resource.
	ConfigFilePath string `json:"-" bundle:"readonly"`

	// Relative path from the bundle root to the configuration file that holds
	// the definition of this resource.
	RelativePath string `json:"relative_path,omitempty" bundle:"readonly"`
}

func (p *Paths) ConfigFileDirectory() (string, error) {
	if p.ConfigFilePath == "" {
		return "", fmt.Errorf("config file path not configured")
	}
	return filepath.Dir(p.ConfigFilePath), nil
}
