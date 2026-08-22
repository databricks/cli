// Package bundle is the top level package for Declarative Automation Bundles.
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
	"time"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cache"
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

// QuietLevel is how much of the informational output to suppress, controlled by
// repeating -q. Warnings and errors are never suppressed.
type QuietLevel int

const (
	// QuietNone prints everything.
	QuietNone QuietLevel = iota

	// QuietSummary (-q) drops the per-resource action lines, keeping the summary.
	QuietSummary

	// QuietAll (-qq) also drops the summary and the progress lines ("Uploading
	// bundle files to ...", "Building ...", "Executing 'postdeploy' script"), so
	// only warnings and errors remain.
	QuietAll
)

// SuppressProgress reports whether progress and summary output should be skipped.
func (b *Bundle) SuppressProgress() bool {
	return b.Quiet >= QuietAll
}

// AiCodeSnapshotDir is the sync-relative dir the aicode mutator writes AI Runtime
// code snapshots into. Force-included in sync (see GetSyncIncludePatterns) so user
// ignore rules can't filter the deployed job's code_source_path archives out.
const AiCodeSnapshotDir = ".air_snapshots"

// Filename where resources are stored for DATABRICKS_BUNDLE_ENGINE=direct
const resourcesFilename = "resources.json"

// Filename where resources are stored for DATABRICKS_BUNDLE_ENGINE=terraform
const terraformStateFilename = "terraform.tfstate"

// Filename where config snapshot is stored for experimental YAML sync
const configSnapshotFilename = "resources-config-sync-snapshot.json"

// This struct is used as a communication channel to collect metrics
// from all over the bundle codebase to finally be emitted as telemetry.
type Metrics struct {
	ConfigurationFileCount      int64
	TargetCount                 int64
	DeploymentId                uuid.UUID
	BoolValues                  []protos.BoolMapEntry
	PythonAddedResourcesCount   int64
	PythonUpdatedResourcesCount int64
	ExecutionTimes              []protos.IntMapEntry
	LocalCacheMeasurementsMs    []protos.IntMapEntry // Local cache measurements stored as milliseconds

	// StateEngine is the engine that ran the deploy, set in deployCore. Empty when
	// telemetry is emitted without a deploy having run.
	StateEngine engine.EngineType

	// ResourceState is the direct engine's per-resource deployment state
	// captured right after the deploy. It carries each resource's state-size in
	// bytes so deploy telemetry can be derived without re-reading or re-parsing
	// the state file. Nil for terraform deploys.
	ResourceState resourcestate.ExportedResourcesMap
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

// AddDurationValue sets the value of a duration metric in milliseconds.
// The value is added to the list of measurements.
func (m *Metrics) AddDurationValue(key string, value time.Duration) {
	valueMs := value.Milliseconds()
	m.LocalCacheMeasurementsMs = append(m.LocalCacheMeasurementsMs, protos.IntMapEntry{Key: key, Value: valueMs})
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

	// includePatterns holds the raw (unexpanded) 'include' patterns from the root
	// databricks.yml. ProcessRootIncludes overwrites Config.Include with the
	// expanded list of loaded files, so this preserves the original patterns for
	// IsFileIncluded. Set via SetIncludePatterns.
	includePatterns []string

	// Target stores a snapshot of the Root.Bundle.Target configuration when it was selected by SelectTarget.
	Target *config.Target `json:"target_config,omitempty" bundle:"internal"`

	// Metadata about the bundle deployment. This is the interface Databricks services
	// rely on to integrate with bundles when they need additional information about
	// a bundle deployment.
	//
	// After deploy, a file containing the metadata (metadata.json) can be found
	// in the WSFS location containing the bundle state.
	Metadata metadata.Metadata

	// Returns the workspace client, initializing it on first call.
	getClient func() (*databricks.WorkspaceClient, error)

	// Files that are synced to the workspace.file_path
	Files []fileset.File

	// FileCounts is how many files the deploy uploaded and deleted. Unlike Files,
	// which lists everything tracked, this counts only what actually changed.
	FileCounts libsync.FileCounts

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

	// Select contains resource selectors passed via --select flag.
	// When non-empty, only the specified resources are included in deployment.
	Select []string

	// MigratingToDirect is set when the direct engine is requested but the existing
	// state still uses terraform, so the state is migrated to the direct engine after
	// this deploy. Resources that only the direct engine supports are skipped by this
	// run rather than rejected: terraform cannot deploy them, and since terraform
	// could never have deployed them they are absent from its state. The next deploy,
	// which runs on the migrated state, creates them.
	MigratingToDirect bool

	// Quiet is the output verbosity reduction requested via -q/--quiet, which is
	// repeatable: QuietSummary drops the per-resource lines, QuietAll additionally
	// drops the summary and progress lines, leaving warnings and errors.
	Quiet QuietLevel

	// SkipLocalFileValidation makes path translation tolerant of missing local files.
	// When set, TranslatePaths computes workspace paths without verifying files exist.
	// Used by config-remote-sync: a user may modify resource paths remotely (e.g.,
	// rename a pipeline root folder in the UI), and the updated paths may not exist
	// locally. Path translation is still needed to produce fully resolved paths for
	// comparison with remote state, but local file validation would incorrectly fail.
	SkipLocalFileValidation bool

	// HasAiRuntimeCodeSnapshot is set by the aicode.PackageCodeSource build-phase
	// mutator when it packages a local AI Runtime code_source into the bundle's
	// snapshot dir. GetSyncIncludePatterns reads it to force-sync that dir only for
	// bundles that actually use the feature, rather than for every bundle.
	HasAiRuntimeCodeSnapshot bool

	// Tagging is used to normalize tag keys and values.
	// The implementation depends on the cloud being targeted.
	Tagging tags.Cloud

	// Cache is used for caching API responses (e.g., current user).
	// By default, caching is enabled. Set DATABRICKS_CACHE_ENABLED=false to disable caching.
	Cache *cache.Cache

	Metrics Metrics

	// Telemetry mirrors the boolean telemetry flags into a typed struct for the
	// bundle bitmap. It is additive: the Metrics mechanism above still feeds the
	// telemetry wire unchanged.
	Telemetry Telemetry
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
	root, diags := mustGetRoot(ctx)
	if diags.HasError() {
		for _, d := range diags {
			logdiag.LogDiag(ctx, d)
		}
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
	root, diags := tryGetRoot(ctx)
	if diags.HasError() {
		for _, d := range diags {
			logdiag.LogDiag(ctx, d)
		}
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

func (b *Bundle) initClientOnce(ctx context.Context) {
	b.getClient = sync.OnceValues(func() (*databricks.WorkspaceClient, error) {
		w, err := b.Config.Workspace.Client(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve bundle auth configuration: %w", err)
		}
		return w, nil
	})
}

func (b *Bundle) WorkspaceClientE(ctx context.Context) (*databricks.WorkspaceClient, error) {
	if b.getClient == nil {
		b.initClientOnce(ctx)
	}
	return b.getClient()
}

func (b *Bundle) WorkspaceClient(ctx context.Context) *databricks.WorkspaceClient {
	client, err := b.WorkspaceClientE(ctx)
	if err != nil {
		panic(err)
	}

	return client
}

// SetWorkpaceClient sets the workspace client for this bundle.
// This is used to inject a mock client for testing.
func (b *Bundle) SetWorkpaceClient(w *databricks.WorkspaceClient) {
	b.getClient = func() (*databricks.WorkspaceClient, error) {
		return w, nil
	}
}

// ClearWorkspaceClient resets the workspace client cache, allowing
// WorkspaceClientE() to attempt client creation again on the next call.
func (b *Bundle) ClearWorkspaceClient(ctx context.Context) {
	b.initClientOnce(ctx)
}

// LocalStateDir returns directory to use for temporary files for this bundle without creating
// Scoped to the bundle's target.
func (b *Bundle) GetLocalStateDir(ctx context.Context, paths ...string) string {
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
	return dir
}

// LocalStateDir returns directory to use for temporary files for this bundle.
// Directory is created and initialized with .gitignore
// Scoped to the bundle's target.
func (b *Bundle) LocalStateDir(ctx context.Context, paths ...string) (string, error) {
	dir := b.GetLocalStateDir(ctx, paths...)
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
	includes := append(b.Config.Sync.Include, filepath.ToSlash(filepath.Join(internalDirRel, "*.*")))
	// Force-sync generated AI Runtime code snapshots so a user ignore rule (e.g.
	// "*.tar.gz" in .gitignore) can't filter them out — the deployed job's
	// code_source_path points at these archives (see bundle/config/mutator/aicode).
	// Scoped to bundles that actually package one, so it's not a global include.
	if b.HasAiRuntimeCodeSnapshot {
		includes = append(includes, AiCodeSnapshotDir+"/*")
	}
	return includes, nil
}

// AuthEnv returns a map with environment variables and their values
// derived from the workspace client configuration that was resolved
// in the context of this bundle.
//
// This map can be used to configure authentication for tools that
// we call into from this bundle context.
func (b *Bundle) AuthEnv(ctx context.Context) (map[string]string, error) {
	w, err := b.WorkspaceClientE(ctx)
	if err != nil {
		return nil, err
	}

	return auth.Env(w.Config), nil
}

// StateFilenameDirect returns (relative remote path, relative local path) for direct engine resource state
func (b *Bundle) StateFilenameDirect(ctx context.Context) (string, string) {
	return resourcesFilename, filepath.ToSlash(filepath.Join(b.GetLocalStateDir(ctx), resourcesFilename))
}

func (b *Bundle) StateFilenameTerraform(ctx context.Context) (string, string) {
	return terraformStateFilename, filepath.ToSlash(filepath.Join(b.GetLocalStateDir(ctx), "terraform", terraformStateFilename))
}

// StateFilenameConfigSnapshot returns (relative remote path, relative local path) for config snapshot state
func (b *Bundle) StateFilenameConfigSnapshot(ctx context.Context) (string, string) {
	return configSnapshotFilename, filepath.ToSlash(filepath.Join(b.GetLocalStateDir(ctx), configSnapshotFilename))
}

// IsImmutableFolder reports whether experimental.immutable_folder is enabled.
func (b *Bundle) IsImmutableFolder() bool {
	return b.Config.Experimental != nil && b.Config.Experimental.ImmutableFolder
}
