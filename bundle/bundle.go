// Package bundle is the top level package for Databricks Asset Bundles.
//
// A bundle is represented by the [Bundle] type. It consists of configuration
// and runtime state, such as a client to a Databricks workspace.
// Every mutation to a bundle's configuration or state is represented as a [Mutator].
// This interface makes every mutation observable and lets us reason about sequencing.
package bundle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/locker"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	libsync "github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-exec/tfexec"
)

const internalFolder = ".internal"

// Filename where resources are stored for DATABRICKS_BUNDLE_ENGINE=direct
const resourcesFilename = "resources.json"

// Filename where resources are stored for DATABRICKS_BUNDLE_ENGINE=terraform
const terraformStateFilename = "terraform.tfstate"

// This struct is used as a communication channel to collect metrics
// from all over the bundle codebase to finally be emitted as telemetry.
type Metrics struct {
	ConfigurationFileCount          int64
	TargetCount                     int64
	DeploymentId                    uuid.UUID
	BoolValues                      []protos.BoolMapEntry
	PythonAddedResourcesCount       int64
	PythonUpdatedResourcesCount     int64
	JavaScriptAddedResourcesCount   int64
	JavaScriptUpdatedResourcesCount int64
	ExecutionTimes                  []protos.IntMapEntry
}

// SetBoolValue sets the value of a boolean metric.
// If the metric does not exist, it is created.
// If the metric exists, it is updated.
// Ensures that the metric is unique
func (m *Metrics) SetBoolValue(key string, value bool) {
	for i, v := range m.BoolValues {
		if v.Key == key {
			m.BoolValues[i].Value = value
			return
		}
	}
	m.BoolValues = append(m.BoolValues, protos.BoolMapEntry{Key: key, Value: value})
}

func (m *Metrics) AddBoolValue(key string, value bool) {
	m.BoolValues = append(m.BoolValues, protos.BoolMapEntry{Key: key, Value: value})
}

type Bundle struct {
	// BundleRootPath is the local path to the root directory of the bundle.
	// It is set when we instantiate a new bundle instance.
	BundleRootPath string

	// BundleRoot is a virtual filesystem path to [BundleRootPath].
	// Exclusively use this field for filesystem operations.
	BundleRoot vfs.Path

	// SyncRootPath is the local path to the root directory of files that are synchronized to the workspace.
	// By default, it is the same as [BundleRootPath].
	// If it is different, it must be an ancestor to [BundleRootPath].
	// That is, [SyncRootPath] must contain [BundleRootPath].
	SyncRootPath string

	// SyncRoot is a virtual filesystem path to [SyncRootPath].
	// Exclusively use this field for filesystem operations.
	SyncRoot vfs.Path

	// Path to the root of git worktree containing the bundle.
	// https://git-scm.com/docs/git-worktree
	WorktreeRoot vfs.Path

	// Config contains the bundle configuration.
	// It is loaded from the bundle configuration files and mutators may update it.
	Config config.Root

	// Target stores a snapshot of the Root.Bundle.Target configuration when it was selected by SelectTarget.
	Target *config.Target `json:"target_config,omitempty" bundle:"internal"`

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
	clientErr  error

	// Files that are synced to the workspace.file_path
	Files []fileset.File

	// Stores an initialized copy of this bundle's Terraform wrapper.
	Terraform *tfexec.Terraform

	// Stores the locker responsible for acquiring/releasing a deployment lock.
	Locker *locker.Locker

	// TerraformPlanPath is the path to the plan from the terraform CLI
	TerraformPlanPath string

	// If true, the plan is empty and applying it will not do anything
	TerraformPlanIsEmpty bool

	// (direct only) deployment implementation and state
	DeploymentBundle direct.DeploymentBundle

	// if true, we skip approval checks for deploy, destroy resources and delete
	// files
	AutoApprove bool

	// Tagging is used to normalize tag keys and values.
	// The implementation depends on the cloud being targeted.
	Tagging tags.Cloud

	Metrics Metrics

	// If true, don't use terraform. Set by DATABRICKS_BUNDLE_ENGINE=direct
	DirectDeployment bool
}

func Load(ctx context.Context, path string) (*Bundle, error) {
	b := &Bundle{
		BundleRootPath: filepath.Clean(path),
		BundleRoot:     vfs.MustNew(path),
	}
	configFile, err := config.FileNames.FindInPath(path)
	if err != nil {
		return nil, err
	}
	log.Debugf(ctx, "Found bundle root at %s (file %s)", b.BundleRootPath, configFile)
	return b, nil
}

// MustLoad returns a bundle configuration.
// The errors are recorded by logdiag, check with logdiag.HasError().
func MustLoad(ctx context.Context) *Bundle {
	root, err := mustGetRoot(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}

	logdiag.SetRoot(ctx, root)

	b, err := Load(ctx, root)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}
	return b
}

// TryLoad returns a bundle configuration if there is one, but doesn't fail if there isn't one.
// The errors are recorded by logdiag, check with logdiag.HasError().
// It returns a `nil` bundle if a bundle was not found.
func TryLoad(ctx context.Context) *Bundle {
	root, err := tryGetRoot(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}

	// No root is fine in this function.
	if root == "" {
		return nil
	}

	logdiag.SetRoot(ctx, root)

	b, err := Load(ctx, root)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}
	return b
}

func (b *Bundle) WorkspaceClientE() (*databricks.WorkspaceClient, error) {
	b.clientOnce.Do(func() {
		var err error
		b.client, err = b.Config.Workspace.Client()
		if err != nil {
			b.clientErr = fmt.Errorf("cannot resolve bundle auth configuration: %w", err)
		}
	})

	return b.client, b.clientErr
}

func (b *Bundle) WorkspaceClient() *databricks.WorkspaceClient {
	client, err := b.WorkspaceClientE()
	if err != nil {
		panic(err)
	}

	return client
}

// SetWorkpaceClient sets the workspace client for this bundle.
// This is used to inject a mock client for testing.
func (b *Bundle) SetWorkpaceClient(w *databricks.WorkspaceClient) {
	b.clientOnce.Do(func() {})
	b.client = w
}

// LocalStateDir returns directory to use for temporary files for this bundle.
// Scoped to the bundle's target.
func (b *Bundle) LocalStateDir(ctx context.Context, paths ...string) (string, error) {
	if b.Config.Bundle.Target == "" {
		panic("target not set")
	}

	cacheDirName, exists := env.TempDir(ctx)
	if !exists || cacheDirName == "" {
		cacheDirName = filepath.Join(
			// Anchor at bundle root directory.
			b.BundleRootPath,
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
	err := os.MkdirAll(dir, 0o700)
	if err != nil {
		return "", err
	}

	libsync.WriteGitIgnore(ctx, b.BundleRootPath)
	return dir, nil
}

// This directory is used to store and automaticaly sync internal bundle files, such as, f.e
// notebook trampoline files for Python wheel and etc.
func (b *Bundle) InternalDir(ctx context.Context) (string, error) {
	cacheDir, err := b.LocalStateDir(ctx)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(cacheDir, internalFolder)
	err = os.MkdirAll(dir, 0o700)
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
	internalDirRel, err := filepath.Rel(b.BundleRootPath, internalDir)
	if err != nil {
		return nil, err
	}
	return append(b.Config.Sync.Include, filepath.ToSlash(filepath.Join(internalDirRel, "*.*"))), nil
}

// AuthEnv returns a map with environment variables and their values
// derived from the workspace client configuration that was resolved
// in the context of this bundle.
//
// This map can be used to configure authentication for tools that
// we call into from this bundle context.
func (b *Bundle) AuthEnv() (map[string]string, error) {
	if b.client == nil {
		return nil, errors.New("workspace client not initialized yet")
	}

	cfg := b.client.Config
	return auth.Env(cfg), nil
}

func (b *Bundle) StateFilename() string {
	if b.DirectDeployment {
		return resourcesFilename
	} else {
		return terraformStateFilename
	}
}

func (b *Bundle) StateLocalPath(ctx context.Context) (string, error) {
	if b.DirectDeployment {
		cacheDir, err := b.LocalStateDir(ctx)
		if err != nil {
			return "", err
		}
		return filepath.Join(cacheDir, resourcesFilename), nil
	} else {
		cacheDir, err := b.LocalStateDir(ctx, "terraform")
		if err != nil {
			return "", err
		}
		return filepath.Join(cacheDir, terraformStateFilename), nil
	}
}

func (b *Bundle) OpenStateFile(ctx context.Context) error {
	if !b.DirectDeployment {
		panic("internal error: OpenResourceDatabase must be called with DirectDeployment")
	}

	statePath, err := b.StateLocalPath(ctx)
	if err != nil {
		return err
	}

	err = b.DeploymentBundle.OpenStateFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to open/create state file at %s: %s", statePath, err)
	}

	return nil
}
