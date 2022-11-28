package bundle

import (
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

	localRoot           string
	remoteRoot          string
	env                 string
	locker              *DeployLocker
	user                string
	terraformBinaryPath string

	client *databricks.WorkspaceClient
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
