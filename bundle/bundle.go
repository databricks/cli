package bundle

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/databricks/databricks-sdk-go"
)

type Bundle struct {
	Config config.Root

	// Store a pointer to the workspace client.
	// It can be initialized on demand after loading the configuration.
	clientOnce sync.Once
	client     *databricks.WorkspaceClient
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

func (b *Bundle) WorkspaceClient() *databricks.WorkspaceClient {
	b.clientOnce.Do(func() {
		var err error
		b.client, err = b.Config.Workspace.Client()
		if err != nil {
			panic(err)
		}
	})
	return b.client
}
