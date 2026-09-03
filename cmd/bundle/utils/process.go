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
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
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

	// If true, configure outputHandler for phases.Deploy
	Verbose bool

	// If true, call corresponding phase:
	FastValidate    bool
	Validate        bool
	Build           bool
	PreDeployChecks bool
	Deploy          bool

	// Path to pre-computed plan JSON file (direct engine only).
	// When set, skips Build and PreDeployChecks phases, loads plan from file instead of calculating.
	ReadPlanPath string

	// PostStateFunc is called at the end of ProcessBundleRet, within the state lifecycle scope
	// (after state is opened and IDs loaded, before deferred Finalize).
	PostStateFunc func(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) error

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

	// The current deployment read from the service (nil, id "" if there is none yet). Used for the
	// metadata diff and to reject a saved plan that predates the deployment's recorded version.
	var dmsDeployment *bundledeployments.Deployment
	var dmsDeploymentID string

	shouldReadState := opts.ReadState || opts.AlwaysPull || opts.InitIDs || opts.ErrorOnEmptyState || opts.PreDeployChecks || opts.Deploy || opts.ReadPlanPath != ""

	if shouldReadState {
		// PullResourcesState depends on stateFiler which needs b.Config.Workspace.StatePath which is set in phases.Initialize
		ctx, stateDesc = statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(opts.AlwaysPull), requiredEngine)
		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
		cmd.SetContext(ctx)

		b.MigratingToDirect = requiredEngine.Type == engine.EngineDirect && !stateDesc.Engine.IsDirect()

		// Announce the auto-migration path here (only on deploy) so the user
		// isn't surprised when MigrateToDirect commits state changes at the
		// end. PullResourcesState is shared with non-deploy commands like
		// `bundle debug states`, which would otherwise print the same hint
		// even though they will not migrate.
		if opts.Deploy && b.MigratingToDirect {
			if requiredEngine.IsDefault {
				// The user did not ask for direct; it is the default. Frame the
				// auto-migration as an informational notice rather than a warning,
				// and do not claim the user selected anything.
				cmdio.LogString(ctx, "Notice: the direct deployment engine is the default as of CLI v1.14.0.\n\n"+
					"This bundle will be automatically migrated to use the direct deployment engine after this deployment.\n\n"+
					"Learn more: https://docs.databricks.com/dev-tools/bundles/direct\n")
			} else {
				log.Warnf(ctx, "Direct engine selected via %s but the existing state uses %q. Deploying on %q; will attempt to migrate the state to the direct engine after this deploy.", requiredEngine.Source, stateDesc.Engine, stateDesc.Engine)
			}
		}

		// --select is only supported by the direct engine, which tracks resource
		// dependencies in the plan graph (used to expand the selection transitively).
		// The engine is only known for certain after the state is pulled, so reject it
		// here rather than silently planning/deploying every resource on terraform.
		if len(b.Select) > 0 && !stateDesc.Engine.IsDirect() {
			logdiag.LogError(ctx, errors.New("--select is only supported with the direct engine. See https://docs.databricks.com/aws/en/dev-tools/bundles/direct"))
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		// Open direct engine state once for all subsequent operations (ExportState, CalculatePlan, Apply, etc.)
		needDirectState := stateDesc.Engine.IsDirect() && (opts.InitIDs || opts.ErrorOnEmptyState || opts.Deploy || opts.ReadPlanPath != "" || opts.PreDeployChecks || opts.PostStateFunc != nil)
		if needDirectState {
			_, localPath := b.StateFilenameDirect(ctx)

			if b.ConfiguresDeploymentHistory(ctx) {
				deploymentID, deployment, err := fetchDeploymentFromStatePath(ctx, b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
				if err != nil {
					logdiag.LogError(ctx, err)
					return b, stateDesc, root.ErrAlreadyPrinted
				}

				dmsDeploymentID = deploymentID
				dmsDeployment = deployment

				// Stamp the deployment and the version this run records onto every job and pipeline so
				// the plan carries them. version_id is always known (last recorded + 1); deployment_id
				// does not exist until a first deploy creates it, so it is left off here and the deploy
				// phase stamps the created id.
				lastVersionID := ""
				if deployment != nil {
					lastVersionID = deployment.LastVersionId
				}
				nextVersion, verr := dms.NextVersion(lastVersionID)
				if verr != nil {
					logdiag.LogError(ctx, verr)
					return b, stateDesc, root.ErrAlreadyPrinted
				}
				muts := []bundle.Mutator{metadata.AnnotateDeploymentVersion(nextVersion)}
				if deploymentID != "" {
					bundle.ApplyFuncContext(ctx, b, func(_ context.Context, b *bundle.Bundle) {
						b.Config.Bundle.Deployment.History = &config.DeploymentHistory{
							DeploymentID:    deploymentID,
							LatestVersionID: deployment.LastVersionId,
						}
					})
					muts = append(muts, metadata.AnnotateDeployment(deploymentID))
				}
				bundle.ApplySeqContext(ctx, b, muts...)
				if logdiag.HasError(ctx) {
					return b, stateDesc, root.ErrAlreadyPrinted
				}
			}
			if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false), b.WorkspaceClient(ctx), dstate.WithDeploymentHistory(b.ConfiguresDeploymentHistory(ctx)), dmsDeploymentID); err != nil {
				logdiag.LogError(ctx, err)
				return b, stateDesc, root.ErrAlreadyPrinted
			}

			// Open built the DMS client from the workspace client when recording; hand it to the
			// phases that create versions and deployments.
			if b.ConfiguresDeploymentHistory(ctx) {
				b.DeploymentBundle.DmsApiClient = b.DeploymentBundle.StateDB.DmsClient()
			}

			// The service holds this deployment, so it has to keep being recorded: deploying
			// without recording would leave it describing resources that have moved on.
			if !b.ConfiguresDeploymentHistory(ctx) && b.DeploymentBundle.StateDB.RequiresDeploymentHistory() {
				logdiag.LogError(ctx, errors.New(`unsetting experimental.record_deployment_history is not supported

This deployment's resources are recorded with the deployment metadata service. Set experimental.record_deployment_history: true to deploy or destroy this bundle`))
				return b, stateDesc, root.ErrAlreadyPrinted
			}

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

	}

	var plan *deployplan.Plan

	if opts.ReadPlanPath != "" {
		if !stateDesc.Engine.IsDirect() {
			logdiag.LogError(ctx, errors.New("--plan is only supported with direct engine (set bundle.engine to \"direct\" or DATABRICKS_BUNDLE_ENGINE=direct)"))
			return b, stateDesc, root.ErrAlreadyPrinted
		}
		opts.Build = false
		opts.PreDeployChecks = false

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

		// The plan records the DMS deployment and version it targeted. Reject it if the live
		// deployment moved on - a newer version (someone deployed since, possibly from another
		// machine) or a different id (deleted and recreated) - so a stale plan is never applied on
		// top of newer state. This runs before the lineage/serial check and is the authoritative
		// stale guard for recorded bundles, whose local state is only a tombstone (its serial does
		// not catch a deploy from elsewhere). Both sides are empty when not recording, a no-op there.
		remoteLastVersion := ""
		if dmsDeployment != nil {
			remoteLastVersion = dmsDeployment.LastVersionId
		}
		if plan.LastVersionId != remoteLastVersion {
			logdiag.LogError(ctx, fmt.Errorf("this plan predates the deployment's current version %s; run 'bundle plan' again", remoteLastVersion))
			return b, stateDesc, root.ErrAlreadyPrinted
		}
		if plan.DeploymentId != dmsDeploymentID {
			logdiag.LogError(ctx, errors.New("this plan targets a different deployment than the one now recorded for this bundle; run 'bundle plan' again"))
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		// Validate that the plan's lineage and serial match the local state. This is the stale guard
		// for non-recorded bundles (recorded ones are covered by the version check above).
		err = direct.ValidatePlanAgainstState(&b.DeploymentBundle.StateDB, plan)
		if err != nil {
			logdiag.LogError(ctx, err)
			return b, stateDesc, root.ErrAlreadyPrinted
		}
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

	if opts.PreDeployChecks {
		downgradeWarningToError := !opts.Deploy
		phases.PreDeployChecks(ctx, b, downgradeWarningToError, stateDesc.Engine)

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

		if b != nil && stateDesc != nil && stateDesc.Engine.IsDirect() && stateDesc.HasRemoteTerraformState() {
			statemgmt.BackupRemoteTerraformState(ctx, b)

			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}
	}

	if opts.PostStateFunc != nil {
		if err := opts.PostStateFunc(ctx, b, stateDesc); err != nil {
			return b, stateDesc, err
		}
	}

	return b, stateDesc, nil
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
