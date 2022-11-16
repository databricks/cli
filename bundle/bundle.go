package bundle

import (
	"path/filepath"

	"github.com/databricks/bricks/bundle/config"
)

// ConfigFile is the name of bundle configuration file.
const ConfigFile = "bundle.yml"

type Bundle struct {
	Config config.Root
}

func Load(path string) (*Bundle, error) {
	bundle := &Bundle{
		Config: config.Root{
			Path: path,
		},
	}
	err := bundle.Config.Load(filepath.Join(path, ConfigFile))
	if err != nil {
		return nil, err
	}
	return bundle, nil
}
