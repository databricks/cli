package bundle

import (
	"path/filepath"

	"github.com/databricks/bricks/bundle/config"
)

type Bundle struct {
	Config config.Root
}

func Load(path string) (*Bundle, error) {
	bundle := &Bundle{
		Config: config.Root{
			Path: path,
		},
	}
	err := bundle.Config.Load(filepath.Join(path, config.FileName))
	if err != nil {
		return nil, err
	}
	return bundle, nil
}
