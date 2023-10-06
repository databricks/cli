package paths

import (
	"fmt"
	"path/filepath"
)

type Paths struct {
	// ConfigFilePath holds the path to the configuration file that
	// described the resource that this type is embedded in.
	ConfigFilePath string `json:"config_file_path,omitempty" bundle:"readonly"`
}

func (p *Paths) ConfigFileDirectory() (string, error) {
	if p.ConfigFilePath == "" {
		return "", fmt.Errorf("config file path not configured")
	}
	return filepath.Dir(p.ConfigFilePath), nil
}
