package phases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deploy/snapshot"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
)

var deployApprovalGroups = []approvalGroup{
	{group: "schemas", message: deleteOrRecreateSchemaMessage, skipChildren: true},
	{group: "pipelines", message: deleteOrRecreatePipelineMessage},
	{group: "volumes", message: deleteOrRecreateVolumeMessage},
	{group: "dashboards", message: deleteOrRecreateDashboardMessage},
	{group: "database_instances", message: deleteOrRecreateDatabaseInstanceMessage},
	{group: "synced_database_tables", message: deleteOrRecreateSyncedDatabaseTableMessage},
	{group: "postgres_projects", message: deleteOrRecreatePostgresProjectMessage},
	{group: "postgres_branches", message: deleteOrRecreatePostgresBranchMessage},
	{group: "postgres_databases", message: deleteOrRecreatePostgresDatabaseMessage},
	{group: "vector_search_indexes", message: deleteOrRecreateVectorSearchIndexMessage},
	{group: "genie_spaces", message: deleteOrRecreateGenieSpaceMessage},
}

func approvalForDeploy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (bool, error) {
	actions := plan.GetActions()

	// Deletes of resources that are already gone remotely only clean up the state,
	// so they don't count as destructive actions and need no approval.
	actions = slices.DeleteFunc(actions, func(a deployplan.Action) bool { return a.Gone })

	err := checkForPreventDestroy(b, actions)
	if err != nil {
		return false, err
	}

	total := logApprovalGroups(ctx, actions, deployApprovalGroups, false, deployplan.Recreate, deployplan.Delete)
	if total == 0 {
		// No destructive actions in any tracked group: skip the prompt.
		return true, nil
	}

	if b.AutoApprove {
		return true, nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		return false, errors.New("the deployment requires destructive actions, but the current console does not support prompting.\n" +
			DataLossWarning + "\n" +
			"To proceed, use --auto-approve after reviewing the plan above." + agent.AgentNotice())
	}

	cmdio.LogString(ctx, "")
	return cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
}

func deployCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, stateEngine engine.EngineType) {
	// Apply resources and capture post-apply state.
	// For direct: Finalize flushes the WAL to disk and returns the state;
	// called even if Apply failed so partial progress is saved.
	// For terraform: ParseResourcesState reads the file written by terraform.Apply.
	var (
		state statemgmt.ExportedResourcesMap
		err   error
	)
	// ThisOrDefault: an unset setting still runs the default engine, and telemetry
	// should report the engine that ran.
	b.Metrics.StateEngine = stateEngine.ThisOrDefault()
	if stateEngine.IsDirect() {
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan)
		state, err = b.DeploymentBundle.StateDB.Finalize(ctx)
		// Capture the finalized state for deploy telemetry. It carries each
		// resource's state-size in bytes (from the WAL replay Finalize just
		// did), so telemetry needs no extra read or parse of the state file.
		b.Metrics.ResourceState = state
	} else {
		bundle.ApplyContext(ctx, b, terraform.Apply())
		state, err = terraform.ParseResourcesState(ctx, b)
	}
	if err != nil {
		logdiag.LogError(ctx, err)
	}

	// Even if deployment failed, there might be updates in states that we need to upload
	statemgmt.PushResourcesState(ctx, b, stateEngine)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(state),
		metadata.Compute(),
		metadata.Upload(),
		statemgmt.UploadStateForYamlSync(stateEngine),
	)
}

// logFileSummary reports what the file sync did. Separate from the resource summary
// because a deploy that only changes business logic (a .py or .sql file) leaves every
// resource unchanged, so without this line its summary is all zeros and looks like a
// no-op. Called on the failure paths too: the files were uploaded before whatever
// failed afterwards, so the count is accurate even then.
func logFileSummary(ctx context.Context, b *bundle.Bundle) {
	if b.Quiet >= bundle.QuietAll {
		return
	}
	cmdio.LogString(ctx, fmt.Sprintf("Files: %d uploaded, %d deleted", b.FileCounts.Uploaded, b.FileCounts.Deleted))
}

// logDeploySummary prints the per-resource actions that were applied followed by the
// resource summary line. -q drops the per-resource lines, -qq drops the summary too.
// The past-tense verb is the short action name plus "d" (create→Created,
// delete→Deleted, ...), capitalized to match the sentence case of other output.
// "bundle plan" keeps the lower-case present tense, so the two are still
// distinguishable at a glance.
func logDeploySummary(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) {
	if b.Quiet >= bundle.QuietAll {
		return
	}

	if b.Quiet < bundle.QuietSummary {
		for _, action := range plan.GetActions() {
			if action.ActionType == deployplan.Skip || action.ActionType == deployplan.Undefined {
				continue
			}
			verb := action.ActionType.StringShort() + "d"
			cmdio.LogString(ctx, strings.ToUpper(verb[:1])+verb[1:]+" "+strings.TrimPrefix(action.ResourceKey, "resources."))
		}
	}

	logFileSummary(ctx, b)

	counts := plan.CountActions()
	summary := fmt.Sprintf("Resources: %d created, %d changed, %d deleted, %d unchanged", counts.Create, counts.Change, counts.Delete, counts.Unchanged)
	// Gate on the plan's own NotSelected (not b.Select) so the suffix survives a
	// deploy from a --plan file, where --select was applied at plan time and
	// b.Select is empty here. NotSelected is only ever set by FilterToSelected.
	if plan.NotSelected > 0 {
		summary += fmt.Sprintf(", %d not selected", plan.NotSelected)
	}
	cmdio.LogString(ctx, summary)
}

// uploadLibraries uploads libraries to the workspace.
// It also cleans up the artifacts directory and transforms wheel tasks.
// It is called by only "bundle deploy".
func uploadLibraries(ctx context.Context, b *bundle.Bundle, libs map[string][]libraries.LocationToUpdate) {
	bundle.ApplySeqContext(ctx, b,
		artifacts.CleanUp(),
		libraries.Upload(libs),
	)
}

// The deploy phase deploys artifacts and resources.
// If readPlanPath is provided, the plan is loaded from that file instead of being calculated.
// stateEngine is the engine the resolved state file uses; requestedEngine is
// what bundle.engine / DATABRICKS_BUNDLE_ENGINE asked for and may differ (used
// only by the post-deploy migration check).
func Deploy(ctx context.Context, b *bundle.Bundle, outputHandler sync.OutputHandler, stateEngine engine.EngineType, requestedEngine engine.EngineSetting, libs map[string][]libraries.LocationToUpdate, plan *deployplan.Plan) {
	log.Info(ctx, "Phase: deploy")

	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	bundle.ApplySeqContext(ctx, b,
		scripts.Execute(config.ScriptPreDeploy),
		lock.Acquire(lock.GoalDeploy),
	)

	if logdiag.HasError(ctx) {
		// lock is not acquired here
		return
	}

	// lock is acquired here
	//
	// The version is created only after approval; CompleteVersion is deferred before
	// lock.Release and no-ops until then.
	recording, err := newRecording(ctx, b, stateEngine, dms.VersionTypeDeploy)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	defer func() {
		if err := recording.Finish(ctx, !logdiag.HasError(ctx)); err != nil {
			logdiag.LogError(ctx, err)
		}
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDeploy))
	}()

	immutable := b.IsImmutableFolder()
	if immutable && !stateEngine.IsDirect() {
		logdiag.LogError(ctx, errors.New("experimental.immutable_folder is only supported with the direct deployment engine"))
		return
	}

	if immutable {
		// Upload all source files and built artifacts as a single immutable snapshot.
		// snapshot.Upload() sets workspace.snapshot_path; the variable-resolution
		// pass expands ${workspace.snapshot_path} placeholders written by translate_paths.
		bundle.ApplySeqContext(ctx, b,
			snapshot.Upload(),
			mutator.ResolveVariableReferencesOnlyResources("workspace"),
		)
		if !logdiag.HasError(ctx) {
			_, libDiags := libraries.ReplaceWithRemotePath(ctx, b)
			for _, d := range libDiags {
				logdiag.LogDiag(ctx, d)
			}
		}
	} else {
		uploadLibraries(ctx, b, libs)
	}

	if logdiag.HasError(ctx) {
		return
	}

	if !immutable {
		bundle.ApplySeqContext(ctx, b, files.Upload(outputHandler))
		if logdiag.HasError(ctx) {
			return
		}
	}

	// From here on the files are uploaded, so report them however the rest of the
	// deploy turns out. Deferred rather than repeated at each of the returns below, so
	// that a new early return cannot silently drop it. On success logDeploySummary
	// prints this line itself, between the per-resource lines and the resource summary,
	// and sets the flag so the defer does not print it twice.
	filesReported := false
	defer func() {
		if !filesReported {
			logFileSummary(ctx, b)
		}
	}()

	bundle.ApplySeqContext(ctx, b,
		deploy.StateUpdate(),
		deploy.StatePush(),
		permissions.ApplyWorkspaceRootPermissions(),
		metrics.TrackUsedCompute(),
		deploy.ResourcePathMkdir(),
	)

	if logdiag.HasError(ctx) {
		return
	}

	// Settle deployment and version before planning. Plan snapshots the config, so
	// both must be stamped before it is computed. Version itself is created after approval.
	if err := recording.Prepare(ctx); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if recording.Version() != 0 {
		// The deployment ID is stamped earlier, when the state is opened; only the
		// version is new here. A first deploy has no ID until now, so stamp both.
		bundle.ApplySeqContext(ctx, b,
			metadata.AnnotateDeployment(recording.DeploymentID()),
			metadata.AnnotateDeploymentVersion(recording.Version()),
		)
		if logdiag.HasError(ctx) {
			return
		}
	}

	planFromFile := plan != nil
	if plan == nil {
		// State is already open for read by process.go (for direct engine)
		plan = RunPlan(ctx, b, stateEngine)
	}

	// Stop before opening the WAL for write if planning failed. UpgradeToWrite
	// writes a WAL header that only deployCore's Finalize commits or discards;
	// returning past it without finalizing leaves a header-only WAL behind.
	if logdiag.HasError(ctx) {
		return
	}

	if stateEngine.IsDirect() {
		// Upgrade from read (opened by process.go) to write mode
		if err := b.DeploymentBundle.StateDB.UpgradeToWrite(); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	if planFromFile {
		// Initialize DeploymentBundle for applying the loaded plan
		err := b.DeploymentBundle.InitForApply(ctx, b.WorkspaceClient(ctx), plan)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	// InitForApply receives ctx and could log a diagnostic without returning an
	// error, so re-check before deploying. (UpgradeToWrite above takes no ctx and
	// thus cannot log, so the earlier check is enough to guard the WAL open.)
	if logdiag.HasError(ctx) {
		return
	}

	haveApproval, approvalErr := approvalForDeploy(ctx, b, plan)
	if !haveApproval {
		// No version was created, so the deferred CompleteVersion is a no-op and the version
		// number is left for the next deploy. Both the user declining and a console that
		// cannot prompt land here.
		if approvalErr != nil {
			logdiag.LogError(ctx, approvalErr)
			return
		}
		cmdio.LogString(ctx, "Deployment cancelled!")
		return
	}

	// Create the version the plan was stamped with, staging an operation for every resource
	// it touches. Doing it here rather than before the prompt means a declined deploy never
	// claims a version number.
	staged, err := stagedOperations(plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	sink, err := recording.Start(ctx, staged)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	logDeploymentVersion(ctx, b, recording)

	// Record operations under that version, so DMS holds the deployed resource state.
	b.DeploymentBundle.OpSink = sink
	deployCore(ctx, b, plan, stateEngine)

	if logdiag.HasError(ctx) {
		return
	}

	// Report what was deployed, mirroring "bundle plan". Printed before the
	// postdeploy script so the deploy's own report is not interleaved with
	// post-deploy output: the script's lines and the migration's below both follow
	// it, and neither reads as belonging to the deploy. The resources were applied
	// above, so the counts are accurate however the script turns out, and its error
	// still propagates. Earlier failures report the files only, since the plan
	// counts would then describe what was intended rather than what was applied.
	filesReported = true
	logDeploySummary(ctx, b, plan)

	bundle.ApplyContext(ctx, b, scripts.Execute(config.ScriptPostDeploy))

	// Migrate the state to the direct engine, if the user opted in (via
	// bundle.engine or DATABRICKS_BUNDLE_ENGINE) and a dry-run of the migration
	// comes back clean. Without the opt-in, or when the dry-run reports problems,
	// nothing is written: only the outcome is recorded in telemetry, and the
	// deploy is unaffected.
	//
	// Last, after the deploy has reported what it did: this is post-deploy work,
	// and its warnings read as belonging to the deploy if they precede the
	// summary.
	//
	// Gated on the deploy alone, which the early return above already guarantees
	// — not on the postdeploy script. The resources were applied before that
	// script ran, so the state is worth migrating even if it failed, the same
	// reasoning that prints the summary ahead of it.
	if !stateEngine.IsDirect() {
		statemgmt.MigrateToDirect(ctx, b, requestedEngine)
	}
}

func RunPlan(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) *deployplan.Plan {
	if engine.IsDirect() {
		plan, err := b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), &b.Config)
		if err != nil {
			logdiag.LogError(ctx, err)
			return nil
		}
		if len(b.Select) > 0 {
			plan.FilterToSelected(b.Select)
		}
		return plan
	}

	// b.Select is rejected for the terraform engine in ProcessBundleRet, so it is
	// never set here.

	bundle.ApplySeqContext(ctx, b,
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Plan(terraform.PlanGoal("deploy")),
	)

	if logdiag.HasError(ctx) {
		return nil
	}

	tf := b.Terraform
	if tf == nil {
		logdiag.LogError(ctx, errors.New("terraform not initialized"))
		return nil
	}

	plan, err := terraform.ShowPlanFile(ctx, tf, b.TerraformPlanPath)
	if err != nil {
		logdiag.LogError(ctx, err)
		return nil
	}

	for _, group := range b.Config.Resources.AllResources() {
		for rKey := range group.Resources {
			resourceKey := "resources." + group.Description.PluralName + "." + rKey
			if _, ok := plan.Plan[resourceKey]; !ok {
				plan.Plan[resourceKey] = &deployplan.PlanEntry{
					Action: deployplan.Skip,
				}
			}
		}
	}

	return plan
}

// If there are more than 1 thousand of a resource type, do not
// include more resources.
// Since we have a timeout of 3 seconds, we cap the maximum number of IDs
// we send in a single request to have reliable telemetry.
const ResourceIdLimit = 1000
