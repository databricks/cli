package paths

import (
	"fmt"
	"path/filepath"
)

type Paths struct {
	// Absolute path on the local file system to the configuration file that holds
	// the definition of this resource.
	LocalConfigFilePath string `json:"-" bundle:"readonly"`

	// Relative path from the bundle root to the configuration file that holds
	// the definition of this resource.
	ConfigFilePath string `json:"config_file_path,omitempty" bundle:"readonly"`
}

func (p *Paths) ConfigFileDirectory() (string, error) {
	if p.LocalConfigFilePath == "" {
		return "", fmt.Errorf("config file path not configured")
	}
	return filepath.Dir(p.LocalConfigFilePath), nil
}
