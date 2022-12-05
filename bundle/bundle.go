package bundle

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/databricks-sdk-go"
)

type Bundle struct {
	Config config.Root

	// Store a pointer to the workspace client.
	// It can be initialized on demand after loading the configuration.
	clientOnce sync.Once
	client     *databricks.WorkspaceClient
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

var cacheDirName = filepath.Join(".databricks", "bundle")

// CacheDir returns directory to use for temporary files for this bundle.
// Scoped to the bundle's environment.
func (b *Bundle) CacheDir() (string, error) {
	if b.Config.Bundle.Environment == "" {
		panic("environment not set")
	}

	// Make directory if it doesn't exist yet.
	dir := filepath.Join(b.Config.Path, cacheDirName, b.Config.Bundle.Environment)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}

	return dir, nil
}
