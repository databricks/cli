package paths

import (
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/libs/config"
)

type Paths struct {
	// Absolute path on the local file system to the configuration file that holds
	// the definition of this resource.
	ConfigFilePath string `json:"-" bundle:"readonly"`

	// DynamicValue stores the [config.Value] of the containing struct.
	// This assumes that this struct is always embedded.
	DynamicValue config.Value
}

func (p *Paths) ConfigureConfigFilePath() {
	if !p.DynamicValue.IsValid() {
		panic("DynamicValue not set")
	}
	p.ConfigFilePath = p.DynamicValue.Location().File
}

func (p *Paths) ConfigFileDirectory() (string, error) {
	if p.ConfigFilePath == "" {
		return "", fmt.Errorf("config file path not configured")
	}
	return filepath.Dir(p.ConfigFilePath), nil
}
