package utils

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type ProcessOptions struct {
	// If true, do not call logdiag.InitContext(); will panic if logdiag context is not initialized
	SkipInitContext bool

	// Function to call after bundle is loaded but before phases.Initialize() is called
	InitFunc func(b *bundle.Bundle)

	// If true, phases.Initialize() is not called
	SkipInitialize bool

	// If true, call PopulateLocations()
	IncludeLocations bool

	// Function to call after phases.Initialize()
	PostInitFunc func(context context.Context, b *bundle.Bundle) error

	// If true, call PullResourcesState() to read state
	ReadState bool

	// AlwaysPull parameter to PullResourcesState()
	// Implies ReadState
	AlwaysPull bool

	// If true, calls statemgmt.Load() to read the state and update resources with IDs; also calls InitializeURLs()
	// Implies ReadState
	InitIDs bool

	// if true, pass ErrorOnEmptyState to statemgmt.Load
	// Implies ReadState
	ErrorOnEmptyState bool

	// If true, an auto-migration from the terraform engine to the direct engine is
	// committed (resources.json written and pushed, terraform.tfstate backed up) rather
	// than kept in memory. Set by the commands that apply changes to remote resources
	// (deploy, destroy); plan and read-only commands migrate in memory only.
	CommitStateMigration bool

	// If true, configure outputHandler for phases.Deploy
	Verbose bool

	// If true, call corresponding phase:
	FastValidate    bool
	Validate        bool
	Build           bool
	PreDeployChecks bool
	Deploy          bool

	// Path to pre-computed plan JSON file (direct engine only).
	// When set, skips Build and PreDeployChecks phases, and loads the plan from
	// the file instead of calculating it. Artifact uploads are handled directly
	// inside Deploy by reading the remote paths from the plan's new_state and
	// finding the matching local files.
	ReadPlanPath string

	// PostStateFunc is called at the end of ProcessBundleRet, within the state lifecycle scope
	// (after state is opened and IDs loaded, before deferred Finalize).
	PostStateFunc func(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) error

	// If true, deployment history configuration is ignored after state is resolved.
	SkipEnforcingDeploymentHistorySetting bool

	// Indicate whether the bundle operation originates from the pipelines CLI
	IsPipelinesCLI bool
}

func ProcessBundle(cmd *cobra.Command, opts ProcessOptions) (*bundle.Bundle, error) {
	b, _, err := ProcessBundleRet(cmd, opts)
	return b, err
}

func ProcessBundleRet(cmd *cobra.Command, opts ProcessOptions) (b *bundle.Bundle, stateDesc *statemgmt.StateDesc, retErr error) {
	var err error
	ctx := cmd.Context()
	if opts.SkipInitContext {
		if !logdiag.IsSetup(ctx) {
			panic("SkipInitContext=true but InitContext was not called")
		}
	} else {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}

	// Load bundle config and apply target
	b = root.MustConfigureBundle(cmd)

	// Log deploy telemetry on all exit paths. This is a defer to ensure
	// telemetry is logged even when the deploy command fails, for both
	// diagnostic errors and regular Go errors.
	if opts.Deploy {
		defer func() {
			if b == nil {
				return
			}
			errMsg := logdiag.GetFirstErrorSummary(ctx)
			if errMsg == "" && retErr != nil && !errors.Is(retErr, root.ErrAlreadyPrinted) {
				errMsg = retErr.Error()
			}
			phases.LogDeployTelemetry(ctx, b, errMsg)
		}()
	}

	if logdiag.HasError(ctx) {
		return b, nil, root.ErrAlreadyPrinted
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		return b, nil, err
	}

	// Initialize variables by assigning them values passed as command line flags
	configureVariables(cmd, b, variables)

	if b == nil || logdiag.HasError(ctx) {
		return b, nil, root.ErrAlreadyPrinted
	}
	ctx = cmd.Context()

	if opts.InitFunc != nil {
		bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) { opts.InitFunc(b) })
	}

	// InitFunc is where -q is applied, so the quiet context can only be derived
	// afterwards. Progress messages are emitted from mutators that receive only a
	// context, not the bundle, so the level has to travel on the context too.
	if b != nil && b.SuppressProgress() {
		ctx = cmdio.WithQuiet(ctx)
		cmd.SetContext(ctx)
	}

	if !opts.SkipInitialize {
		t0 := time.Now()
		phases.Initialize(ctx, b)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Initialize",
			Value: time.Since(t0).Milliseconds(),
		})
		// not checking error right away here, add locations first
	}

	if b != nil {
		// Include location information in the output if the flag is set.
		if opts.IncludeLocations {
			bundle.ApplyContext(ctx, b, mutator.PopulateLocations())
			if logdiag.HasError(ctx) {
				return b, nil, root.ErrAlreadyPrinted
			}
		}
	}

	if logdiag.HasError(ctx) {
		return b, nil, root.ErrAlreadyPrinted
	}

	if opts.PostInitFunc != nil {
		err := opts.PostInitFunc(ctx, b)
		if err != nil {
			return b, nil, err
		}
	}

	// Resolve engine setting up front so a garbage DATABRICKS_BUNDLE_ENGINE
	// value fails every bundle command instead of only the ones that read
	// state. The resolver is cheap (config lookup + env var read); no reason
	// to gate it on state-touching options.
	requiredEngine, err := ResolveEngineSetting(ctx, b)
	if err != nil {
		return b, nil, err
	}

	// Record the requested engine up front so deploy telemetry reports it even when
	// the deploy fails before the state is pulled (e.g. PullResourcesState itself
	// errors). Refined to the state's engine below once it is known; the two differ
	// only mid-migration, when the deploy runs on the existing state's engine.
	b.Metrics.StateEngine = requiredEngine.Type.ThisOrDefault()

	// The current deployment read from the service (nil, id "" if there is none yet). Used for the
	// metadata diff and to reject a saved plan that predates the deployment's recorded version.
	var dmsDeployment *bundledeployments.Deployment
	var dmsDeploymentID string

	shouldReadState := opts.ReadState || opts.AlwaysPull || opts.InitIDs || opts.ErrorOnEmptyState || opts.PreDeployChecks || opts.Deploy || opts.ReadPlanPath != ""

	if shouldReadState {
		// PullResourcesState depends on stateFiler which needs b.Config.Workspace.StatePath which is set in phases.Initialize
		stateDesc = statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(opts.AlwaysPull), requiredEngine)
		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		b.MigratingToDirect = requiredEngine.Type == engine.EngineDirect && !stateDesc.Engine.IsDirect()

		// Tag the user agent with the engine this run actually uses. On the auto-migration
		// path the engine is only final after the migration runs (below), so leave it unset
		// here and tag there; setting it once rather than appending avoids a stale
		// engine/terraform tag alongside engine/direct.
		if !b.MigratingToDirect {
			ctx = useragent.InContext(ctx, "engine", string(stateDesc.Engine))
		}
		cmd.SetContext(ctx)
		if stateDesc.Engine.IsDirect() {
			resolveDeploymentHistory(ctx, b, stateDesc)
		}

		// Record the engine the resolved state uses now, so deploy telemetry reports
		// it even when the deploy fails or is cancelled before deployCore runs.
		b.Metrics.StateEngine = stateDesc.Engine.ThisOrDefault()
	}

	// --plan applies a precomputed plan, so it skips Build and PreDeployChecks; a plain
	// deploy builds and runs the predeploy checks. These only flip opts (no state access),
	// so they run before phases.Build; the plan file itself is loaded during state
	// resolution after the build, once the engine is known and the state is open.
	if opts.ReadPlanPath != "" {
		opts.Build = false
		opts.PreDeployChecks = false
	} else if opts.Deploy {
		opts.Build = true
		opts.PreDeployChecks = true
	}

	if opts.FastValidate {
		t1 := time.Now()
		bundle.ApplyContext(ctx, b, validate.FastValidate())
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "validate.FastValidate",
			Value: time.Since(t1).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		// Pipeline CLI only validation.
		if opts.IsPipelinesCLI {
			rejectDefinitions(ctx, b)
			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}
	}

	if opts.Validate {
		validate.Validate(ctx, b)
		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
	}

	var libs phases.LibLocationMap

	if opts.Build {
		t2 := time.Now()
		libs = phases.Build(ctx, b)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Build",
			Value: time.Since(t2).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
	}

	// Resolve the deployment state after phases.Build, in one place, so the order reads
	// build → migrate → open → (plan/deploy). Running the migration after the build lets
	// its conversion and plan check see library and ${artifacts.*} references resolved by
	// the build, and record resolved remote paths in the migrated state. Commands without
	// a build phase (destroy, summary, ...) reach here with the build skipped, so migration
	// and state opening still happen in this single block.
	var plan *deployplan.Plan
	if shouldReadState {
		// needsState is narrower than shouldReadState: it drops the pull-only flags
		// (ReadState, AlwaysPull) so read-only consumers like "bundle debug states" pull
		// and display the state without triggering a migration, and adds PostStateFunc for
		// commands that operate on the state (destroy, run). It gates migrating and opening
		// the direct state.
		needsState := opts.InitIDs || opts.ErrorOnEmptyState || opts.Deploy || opts.ReadPlanPath != "" || opts.PreDeployChecks || opts.PostStateFunc != nil

		// Migrate a Terraform state to the direct engine: when the direct engine is
		// requested (the default) and the existing state still uses Terraform, convert it so
		// the run proceeds on the direct engine. deploy/destroy commit the migration
		// (resources.json written and pushed, terraform.tfstate backed up); plan and summary
		// keep it in memory. Deriving commit from opts.Deploy keeps every deploy entry point
		// (bundle, pipelines, apps) consistent. If the migration's plan check fails,
		// MigrateTerraformState leaves the Terraform state intact and the run falls back to
		// the terraform engine. Read-only commands set none of these options and keep
		// reading the Terraform state as-is.
		if b.MigratingToDirect && needsState {
			// deploy defers the commit to the deploy phase (after approval); destroy has no
			// deploy phase to defer to, so it commits up front; other state-reading commands
			// (plan, run) migrate in memory only.
			mode := statemgmt.MigratePlan
			if opts.Deploy {
				mode = statemgmt.MigrateDeploy
			} else if opts.CommitStateMigration {
				mode = statemgmt.MigrateCommit
			}
			if err := migrateTerraformToDirect(ctx, b, stateDesc, requiredEngine, mode); err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}

		// Tag the user agent with the engine this run actually uses. The tag is left unset
		// after the pull on the auto-migration path (see above) because the engine is only
		// final here, after the migration ran, was skipped, or fell back. Setting it once
		// (rather than appending) avoids a stale engine/terraform tag alongside engine/direct.
		if b.MigratingToDirect {
			ctx = useragent.InContext(ctx, "engine", string(stateDesc.Engine))
			cmd.SetContext(ctx)
		}

		// --select is only supported by the direct engine, which tracks resource
		// dependencies in the plan graph (used to expand the selection transitively).
		// Validate once the engine is final (after any migration above), rather than
		// silently planning/deploying every resource on terraform.
		if len(b.Select) > 0 && !stateDesc.Engine.IsDirect() {
			logdiag.LogError(ctx, errors.New("--select is only supported with the direct engine. See https://docs.databricks.com/aws/en/dev-tools/bundles/direct"))
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		// Open direct engine state once for all subsequent operations (ExportState, CalculatePlan, Apply, etc.)
		// A migrated-from-Terraform state is already open (seeded in memory above), so skip the disk open.
		needDirectState := stateDesc.Engine.IsDirect() && needsState
		var localPath string
		if needDirectState && !b.DeploymentBundle.StateDB.IsOpen() {
			_, localPath = b.StateFilenameDirect(ctx)
			if !stateDesc.IsDMS() {
				if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
					logdiag.LogError(ctx, err)
					return b, stateDesc, root.ErrAlreadyPrinted
				}
			}
		}

		if stateDesc.Engine.IsDirect() && !opts.SkipEnforcingDeploymentHistorySetting {
			if err := enforceDeploymentHistorySetting(ctx, b, stateDesc, opts.Deploy || opts.PreDeployChecks); err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}

		if needDirectState && stateDesc.IsDMS() {
			var err error
			dmsDeploymentID, dmsDeployment, err = fetchDeploymentFromStatePath(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
			if err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}

			// Stamp the deployment and the version this run records onto every job and pipeline so
			// the plan carries them. version_id is always known (last recorded + 1); deployment_id
			// does not exist until a first deploy creates it, so it is left off here and the deploy
			// phase stamps the created id.
			lastVersionID, err := parseLastVersionID(dmsDeployment)
			if err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}
			nextVersion := lastVersionID + 1
			muts := []bundle.Mutator{metadata.AnnotateDeploymentVersion(nextVersion)}
			if dmsDeploymentID != "" {
				bundle.ApplyFuncContext(ctx, b, func(_ context.Context, b *bundle.Bundle) {
					b.Config.Bundle.Deployment.DeploymentID = dmsDeploymentID
					b.Config.Bundle.Deployment.LatestVersionID = lastVersionID
				})
				muts = append(muts, metadata.AnnotateDeployment(dmsDeploymentID))
			}
			bundle.ApplySeqContext(ctx, b, muts...)
			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
			// StateDB.Open builds the DMS client from the workspace client on the context.
			if !cmdctx.HasWorkspaceClient(ctx) {
				ctx = cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
			}
			if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: dmsDeploymentID, LastVersionID: lastVersionID}); err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}

		if needDirectState {
			// Warn when the state was last written by a newer CLI than the one
			// running now. The state schema version is a hard gate (dstate.Open
			// rejects a too-new state_version), but a state can be written by a
			// newer CLI that shares this schema; that is allowed, and this only
			// hints that a downgrade may be unintended.
			currentVersion := build.GetInfo().Version
			if stateVersion := b.DeploymentBundle.StateDB.StateCLIVersion(); isNewerVersion(stateVersion, currentVersion) {
				log.Warnf(ctx, "State was last deployed with CLI version %s but current version is %s", stateVersion, currentVersion)
			}
		}

		// These are not safe in plan/deploy because they insert empty config settings for deleted resources.
		if opts.InitIDs || opts.ErrorOnEmptyState {
			var modes []statemgmt.LoadMode
			if opts.ErrorOnEmptyState {
				modes = append(modes, statemgmt.ErrorOnEmptyState)
			}
			var state statemgmt.ExportedResourcesMap
			if stateDesc.Engine.IsDirect() {
				state = b.DeploymentBundle.ExportState(ctx)
			} else {
				var err error
				state, err = terraform.ParseResourcesState(ctx, b)
				if err != nil {
					logdiag.LogError(ctx, err)
					return b, stateDesc, root.ErrAlreadyPrinted
				}
			}
			mutators := []bundle.Mutator{
				statemgmt.Load(state, modes...),
			}
			// InitializeURLs makes an extra API call; only run it when URLs are needed.
			if opts.InitIDs {
				mutators = append(mutators, mutator.InitializeURLs())
			}
			bundle.ApplySeqContext(ctx, b, mutators...)
			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}

		// --plan: the engine is now known and the state is open, so validate and load the
		// precomputed plan. Artifact uploads are handled inside Deploy by extracting remote
		// paths from the plan's new_state and finding the matching local files.
		if opts.ReadPlanPath != "" {
			if !stateDesc.Engine.IsDirect() {
				logdiag.LogError(ctx, errors.New("--plan is only supported with direct engine (set bundle.engine to \"direct\" or DATABRICKS_BUNDLE_ENGINE=direct)"))
				return b, stateDesc, root.ErrAlreadyPrinted
			}
			var err error
			plan, err = deployplan.LoadPlanFromFile(opts.ReadPlanPath)
			if err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}
			currentVersion := build.GetInfo().Version
			if plan.CLIVersion != currentVersion {
				log.Warnf(ctx, "Plan was created with CLI version %s but current version is %s", plan.CLIVersion, currentVersion)
			}

			if err := direct.ValidatePlanAgainstState(&b.DeploymentBundle.StateDB, plan); err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}
	}

	if opts.PreDeployChecks {
		downgradeWarningToError := !opts.Deploy
		phases.PreDeployChecks(ctx, b, downgradeWarningToError, stateDesc.Engine)

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
	}

	// The predeploy script can generate or rewrite files that on_file_change
	// watches, so it has to run before those files are fingerprinted below. It
	// stays ahead of the deployment lock, as it was when phases.Deploy ran it.
	if opts.Deploy {
		bundle.ApplyContext(ctx, b, scripts.Execute(config.ScriptPreDeploy))
		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
	}

	// Fingerprint on_file_change triggers once, after every step that can produce
	// a watched file: build and the predeploy script. Reads the sync root that
	// phases.Initialize resolves, so it is skipped along with it; `bundle deploy
	// --plan` recomputes fingerprints that the loaded plan then overrides with the
	// ones it recorded.
	if !opts.SkipInitialize {
		bundle.ApplyContext(ctx, b, mutator.ResolveJobRunFileTriggers())
		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
	}

	if opts.Deploy {
		var outputHandler sync.OutputHandler
		if opts.Verbose {
			outputHandler = func(ctx context.Context, c <-chan sync.Event) {
				sync.TextOutput(ctx, c, cmd.OutOrStdout())
			}
		}

		t3 := time.Now()
		phases.Deploy(ctx, b, outputHandler, stateDesc.Engine, requiredEngine, libs, plan, dmsDeployment)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Deploy",
			Value: time.Since(t3).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		// A migrating deploy already backed up terraform.tfstate when it committed the
		// converted state above; this handles a plain direct deploy that still finds a
		// lingering remote terraform state.
		if b != nil && stateDesc != nil && stateDesc.Engine.IsDirect() && !b.MigratingToDirect && stateDesc.HasRemoteTerraformState() {
			statemgmt.BackupRemoteTerraformState(ctx, b)

			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}

		// The user opted out of the direct engine (engine: terraform), so no migration ran.
		// Do a throwaway conversion of the just-deployed terraform state to record
		// direct_drymigrate_* telemetry — the fleet-wide "could this bundle migrate?"
		// signal. Runs after the deploy so mutating b.Config during the conversion is
		// harmless, and only when the deploy succeeded on a terraform state.
		if stateDesc != nil && requiredEngine.Type == engine.EngineTerraform && !stateDesc.Engine.IsDirect() {
			statemgmt.DryRunMigrationTelemetry(ctx, b)
		}
	}

	if opts.PostStateFunc != nil {
		if err := opts.PostStateFunc(ctx, b, stateDesc); err != nil {
			return b, stateDesc, err
		}
	}

	return b, stateDesc, nil
}

// migrateTerraformToDirect converts the bundle's Terraform state to the direct engine.
// On success it advances stateDesc and metrics to the direct engine. Any failure before
// the migration commits (parse, conversion, plan check, or push) leaves the Terraform
// state intact (migrated=false) so the caller proceeds on the terraform engine and the
// deploy still runs; only a failure after the commit returns an error. The caller tags the
// user agent with the resolved stateDesc.Engine afterwards.
//
// mode selects how the converted state is committed: MigrateDeploy loads it in memory and
// lets the approved deploy commit it, MigrateCommit (destroy) commits it up front, and
// MigratePlan (plan, run) keeps it in memory only. See MigrateTerraformState.
func migrateTerraformToDirect(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc, requiredEngine engine.EngineSetting, mode statemgmt.MigrateMode) error {
	if requiredEngine.IsDefault {
		cmdio.LogString(ctx, "Notice: automatically migrating your bundle to direct deployment engine (https://github.com/databricks/cli/issues/6765).")
	}
	migrated, err := statemgmt.MigrateTerraformState(ctx, b, requiredEngine, mode)
	if err != nil {
		return fmt.Errorf("migrating Terraform state to the direct engine: %w", err)
	}
	if migrated {
		stateDesc.Engine = engine.EngineDirect
		b.Metrics.StateEngine = engine.EngineDirect
	}
	return nil
}

// ResolveEngineSetting determines the effective engine setting by combining bundle config and env var.
// Priority: bundle.engine config > DATABRICKS_BUNDLE_ENGINE env var > engine.Default.
func ResolveEngineSetting(ctx context.Context, b *bundle.Bundle) (engine.EngineSetting, error) {
	configEngine := b.Config.Bundle.Engine

	if configEngine != engine.EngineNotSet {
		source := "bundle.engine setting"
		v := dyn.GetValue(b.Config.Value(), "bundle.engine")
		if locs := v.Locations(); len(locs) > 0 {
			loc := locs[0]
			source = fmt.Sprintf("bundle.engine setting at %s:%d:%d", filepath.ToSlash(loc.File), loc.Line, loc.Column)
		}
		return engine.EngineSetting{Type: configEngine, Source: source, ConfigType: configEngine}, nil
	}

	envEngine, err := engine.FromEnv(ctx)
	if err != nil {
		return engine.EngineSetting{}, err
	}
	if envEngine != engine.EngineNotSet {
		return engine.EngineSetting{Type: envEngine, Source: engine.EnvVar + " environment variable"}, nil
	}

	return engine.EngineSetting{Type: engine.Default, Source: engine.SourceDefault, IsDefault: true}, nil
}

// Lookup and return the deployment object from ${workspace.state_path}/resources.deployment.json
//
// TODO: a deployment is only usable when both the node and the service's record exist, and a
// half-created one - node present, record missing - blocks the bundle here even though
// CreateDeployment already recovers from it. Move this behind a dms.ReadDeployment(ctx, statePath)
// that returns an empty id and version unless both halves are there, leaving the caller to call
// CreateDeployment to create or finalize it.
//
// TODO: ask the service for a lookup by state path, so this is one round trip rather than two - a
// workspace lookup to turn the node into an id, then a get by that id.
func fetchDeploymentFromStatePath(ctx context.Context, w *databricks.WorkspaceClient, statePath string) (string, *bundledeployments.Deployment, error) {
	nodePath := path.Join(statePath, dms.DeploymentNodeName)

	obj, err := w.Workspace.GetStatusByPath(ctx, nodePath)
	if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("looking up deployment at %s: %w", nodePath, err)
	}
	deploymentID := strconv.FormatInt(obj.ObjectId, 10)
	deployment, err := w.BundleDeployments.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
		Name: dms.DeploymentName(deploymentID),
	})
	if err != nil {
		return "", nil, err
	}
	return deploymentID, deployment, nil
}

// parseLastVersionID parses the deployment's last recorded version, which the service reports
// as a string. It returns 0 when the deployment does not exist yet or has no recorded version.
func parseLastVersionID(dmsDeployment *bundledeployments.Deployment) (int, error) {
	if dmsDeployment == nil || dmsDeployment.LastVersionId == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(dmsDeployment.LastVersionId)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last_version_id %q: %w", dmsDeployment.LastVersionId, err)
	}
	return v, nil
}

// OpenDirectStateForRead opens the direct-engine state database read-only. When the bundle
// records deployment history the local state file is only a tombstone, so the resources are
// read from the deployment metadata service instead.
func OpenDirectStateForRead(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) error {
	_, localPath := b.StateFilenameDirect(ctx)
	if !resolveDeploymentHistory(ctx, b, stateDesc) {
		if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{}); err != nil {
			return err
		}
		return enforceDeploymentHistorySetting(ctx, b, stateDesc, false)
	}

	dmsDeploymentID, dmsDeployment, err := fetchDeploymentFromStatePath(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return err
	}
	lastVersionID, err := parseLastVersionID(dmsDeployment)
	if err != nil {
		return err
	}
	// StateDB.Open builds the DMS client from the workspace client on the context, so ensure one is set.
	if !cmdctx.HasWorkspaceClient(ctx) {
		ctx = cmdctx.SetWorkspaceClient(ctx, b.WorkspaceClient(ctx))
	}
	if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(false), dstate.WithWrite(false), dstate.WithDeploymentHistory(true), dstate.OpenDmsArgs{DeploymentID: dmsDeploymentID, LastVersionID: lastVersionID}); err != nil {
		return err
	}
	return enforceDeploymentHistorySetting(ctx, b, stateDesc, false)
}

func resolveDeploymentHistory(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) bool {
	configured := configuresDeploymentHistory(ctx, b)
	if stateDesc.SourcePath == "" {
		if configured {
			stateDesc.Features = map[string]struct{}{dstate.FeatureDeploymentHistory: {}}
		}
		return configured
	}

	return stateDesc.IsDMS()
}

func enforceDeploymentHistorySetting(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc, requireMatch bool) error {
	if stateDesc.SourcePath == "" {
		return nil
	}
	configured := configuresDeploymentHistory(ctx, b)
	recorded := stateDesc.IsDMS()
	if configured == recorded {
		return nil
	}
	if requireMatch {
		return fmt.Errorf(`deployment history setting (%t) does not match the existing state (%t)

Update experimental.deployment_history to match the existing deployment, or run "databricks bundle destroy" to start over`, configured, recorded)
	}
	if configured {
		return errors.New(`enabling experimental.deployment_history for an existing deployment is not supported

Run "databricks bundle destroy" first, then deploy again with deployment history enabled`)
	}
	log.Warnf(ctx, "Deployment history setting (%t) does not match the existing state (%t). Using the existing state.", configured, recorded)
	return nil
}

func configuresDeploymentHistory(ctx context.Context, b *bundle.Bundle) bool {
	configured := b.Config.Experimental != nil && b.Config.Experimental.DeploymentHistory
	return bundleenv.RecordsDeploymentHistory(ctx, configured)
}

// isNewerVersion reports whether the state's recorded CLI version is strictly
// newer than the running build. Both are bare versions without a leading "v".
// An empty stateVersion (state not written by any CLI yet) or an unparseable
// version returns false, so we never warn on missing or malformed data.
func isNewerVersion(stateVersion, currentVersion string) bool {
	sv := "v" + stateVersion
	cv := "v" + currentVersion
	if !semver.IsValid(sv) || !semver.IsValid(cv) {
		return false
	}
	return semver.Compare(sv, cv) > 0
}

func rejectDefinitions(ctx context.Context, b *bundle.Bundle) {
	if b.Config.Definitions != nil {
		v := dyn.GetValue(b.Config.Value(), "definitions")
		loc := v.Locations()
		filename := "input yaml"
		if len(loc) > 0 {
			filename = filepath.ToSlash(loc[0].File)
		}
		logdiag.LogError(ctx, errors.New(filename+` seems to be formatted for open-source Spark Declarative Pipelines.
Pipelines CLI currently only supports Lakeflow Spark Declarative Pipelines development.
To see an example of a supported pipelines template, create a new Pipelines CLI project with "pipelines init".`))
	}
}
