// Package bundle is the top level package for Databricks Asset Bundles.
//
// A bundle is represented by the [Bundle] type. It consists of configuration
// and runtime state, such as a client to a Databricks workspace.
// Every mutation to a bundle's configuration or state is represented as a [Mutator].
// This interface makes every mutation observable and lets us reason about sequencing.
package bundle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/terraform"
	"github.com/databricks/databricks-sdk-go"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/hashicorp/terraform-exec/tfexec"
)

const internalFolder = ".internal"

type Bundle struct {
	// RootPath contains the directory path to the root of the bundle.
	// It is set when we instantiate a new bundle instance.
	RootPath string

	Config config.Root

	// Metadata about the bundle deployment. This is the interface Databricks services
	// rely on to integrate with bundles when they need additional information about
	// a bundle deployment.
	//
	// After deploy, a file containing the metadata (metadata.json) can be found
	// in the WSFS location containing the bundle state.
	Metadata metadata.Metadata

	// Store a pointer to the workspace client.
	// It can be initialized on demand after loading the configuration.
	clientOnce sync.Once
	client     *databricks.WorkspaceClient

	// Stores an initialized copy of this bundle's Terraform wrapper.
	Terraform *tfexec.Terraform

	// Stores the locker responsible for acquiring/releasing a deployment lock.
	Locker *locker.Locker

	Plan *terraform.Plan

	// if true, we skip approval checks for deploy, destroy resources and delete
	// files
	AutoApprove bool

	// Tagging is used to normalize tag keys and values.
	// The implementation depends on the cloud being targeted.
	Tagging tags.Cloud
}

func Load(ctx context.Context, path string) (*Bundle, error) {
	b := &Bundle{
		RootPath: filepath.Clean(path),
	}
	configFile, err := config.FileNames.FindInPath(path)
	if err != nil {
		return nil, err
	}
	log.Debugf(ctx, "Found bundle root at %s (file %s)", b.RootPath, configFile)
	return b, nil
}

// MustLoad returns a bundle configuration.
// It returns an error if a bundle was not found or could not be loaded.
func MustLoad(ctx context.Context) (*Bundle, error) {
	root, err := mustGetRoot(ctx)
	if err != nil {
		return nil, err
	}

	return Load(ctx, root)
}

// TryLoad returns a bundle configuration if there is one, but doesn't fail if there isn't one.
// It returns an error if a bundle was found but could not be loaded.
// It returns a `nil` bundle if a bundle was not found.
func TryLoad(ctx context.Context) (*Bundle, error) {
	root, err := tryGetRoot(ctx)
	if err != nil {
		return nil, err
	}

	// No root is fine in this function.
	if root == "" {
		return nil, nil
	}

	return Load(ctx, root)
}

func (b *Bundle) InitializeWorkspaceClient() (*databricks.WorkspaceClient, error) {
	client, err := b.Config.Workspace.Client()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve bundle auth configuration: %w", err)
	}
	return client, nil
}

func (b *Bundle) WorkspaceClient() *databricks.WorkspaceClient {
	b.clientOnce.Do(func() {
		var err error
		b.client, err = b.InitializeWorkspaceClient()
		if err != nil {
			panic(err)
		}
	})
	return b.client
}

// SetWorkpaceClient sets the workspace client for this bundle.
// This is used to inject a mock client for testing.
func (b *Bundle) SetWorkpaceClient(w *databricks.WorkspaceClient) {
	b.clientOnce.Do(func() {})
	b.client = w
}

// CacheDir returns directory to use for temporary files for this bundle.
// Scoped to the bundle's target.
func (b *Bundle) CacheDir(ctx context.Context, paths ...string) (string, error) {
	if b.Config.Bundle.Target == "" {
		panic("target not set")
	}

	cacheDirName, exists := env.TempDir(ctx)
	if !exists || cacheDirName == "" {
		cacheDirName = filepath.Join(
			// Anchor at bundle root directory.
			b.RootPath,
			// Static cache directory.
			".databricks",
			"bundle",
		)
	}

	// Fixed components of the result path.
	parts := []string{
		cacheDirName,
		// Scope with target name.
		b.Config.Bundle.Target,
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

// This directory is used to store and automaticaly sync internal bundle files, such as, f.e
// notebook trampoline files for Python wheel and etc.
func (b *Bundle) InternalDir(ctx context.Context) (string, error) {
	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(cacheDir, internalFolder)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return dir, err
	}

	return dir, nil
}

// GetSyncIncludePatterns returns a list of user defined includes
// And also adds InternalDir folder to include list for sync command
// so this folder is always synced
func (b *Bundle) GetSyncIncludePatterns(ctx context.Context) ([]string, error) {
	internalDir, err := b.InternalDir(ctx)
	if err != nil {
		return nil, err
	}
	internalDirRel, err := filepath.Rel(b.RootPath, internalDir)
	if err != nil {
		return nil, err
	}
	return append(b.Config.Sync.Include, filepath.ToSlash(filepath.Join(internalDirRel, "*.*"))), nil
}

func (b *Bundle) GitRepository() (*git.Repository, error) {
	rootPath, err := folders.FindDirWithLeaf(b.RootPath, ".git")
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
