package bundle

import (
	"context"
	"path/filepath"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
)

type Bundle struct {
	Config config.Root
}

func (b *Bundle) MutateForEnvironment(env string) error {
	return mutator.Apply(&b.Config, mutator.DefaultMutatorsForEnvironment(env))
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

func LoadFromRoot() (*Bundle, error) {
	root, err := getRoot()
	if err != nil {
		return nil, err
	}

	return Load(root)
}

func ConfigureForEnvironment(ctx context.Context, env string) (context.Context, error) {
	b, err := LoadFromRoot()
	if err != nil {
		return nil, err
	}

	err = b.MutateForEnvironment(env)
	if err != nil {
		return nil, err
	}

	return Context(ctx, b), nil
}
