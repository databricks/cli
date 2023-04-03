// Package bundle is the top level package for Databricks Application Bundles.
//
// A bundle is represented by the [Bundle] type. It consists of configuration
// and runtime state, such as a client to a Databricks workspace.
// Every mutation to a bundle's configuration or state is represented as a [Mutator].
// This interface makes every mutation observable and lets us reason about sequencing.
package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/folders"
	"github.com/databricks/bricks/libs/git"
	"github.com/databricks/bricks/libs/locker"
	"github.com/databricks/databricks-sdk-go"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type Bundle struct {
	Config config.Root

	// Store a pointer to the workspace client.
	// It can be initialized on demand after loading the configuration.
	clientOnce sync.Once
	client     *databricks.WorkspaceClient

	// Stores an initialized copy of this bundle's Terraform wrapper.
	Terraform *tfexec.Terraform

	// Stores the locker responsible for acquiring/releasing a deployment lock.
	Locker *locker.Locker
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

// MustLoad returns a bundle configuration.
// It returns an error if a bundle was not found or could not be loaded.
func MustLoad() (*Bundle, error) {
	root, err := mustGetRoot()
	if err != nil {
		return nil, err
	}

	return Load(root)
}

// TryLoad returns a bundle configuration if there is one, but doesn't fail if there isn't one.
// It returns an error if a bundle was found but could not be loaded.
// It returns a `nil` bundle if a bundle was not found.
func TryLoad() (*Bundle, error) {
	root, err := tryGetRoot()
	if err != nil {
		return nil, err
	}

	// No root is fine in this function.
	if root == "" {
		return nil, nil
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
func (b *Bundle) CacheDir(paths ...string) (string, error) {
	if b.Config.Bundle.Environment == "" {
		panic("environment not set")
	}

	// Fixed components of the result path.
	parts := []string{
		// Anchor at bundle root directory.
		b.Config.Path,
		// Static cache directory.
		cacheDirName,
		// Scope with environment name.
		b.Config.Bundle.Environment,
	}

	// Append dynamic components of the result path.
	parts = append(parts, paths...)

	// Make directory if it doesn't exist yet.
	dir := filepath.Join(parts...)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func (b *Bundle) GitRepository() (*git.Repository, error) {
	rootPath, err := folders.FindDirWithLeaf(b.Config.Path, ".git")
	if err != nil {
		return nil, fmt.Errorf("unable to locate repository root: %w", err)
	}

	return git.NewRepository(rootPath)
}

// AuthEnv returns a map with environment variables and their values
// derived from the workspace client configuration that was resolved
// in the context of this bundle.
//
// This map can be used to configure authentication for tools that
// we call into from this bundle context.
func (b *Bundle) AuthEnv() (map[string]string, error) {
	if b.client == nil {
		return nil, fmt.Errorf("workspace client not initialized yet")
	}

	cfg := b.client.Config
	out := make(map[string]string)
	for _, attr := range sdkconfig.ConfigAttributes {
		// Ignore profile so that downstream tools don't try and reload
		// the profile even though we know the current configuration is valid.
		if attr.Name == "profile" {
			continue
		}
		if len(attr.EnvVars) == 0 {
			continue
		}
		if attr.IsZero(cfg) {
			continue
		}
		out[attr.EnvVars[0]] = attr.GetString(cfg)
	}

	return out, nil
}
