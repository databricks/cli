package phases

import (
	"context"
	"errors"

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

func deployCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, targetEngine engine.EngineType) {
	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	cmdio.LogString(ctx, "Deploying resources...")

	// Apply resources and capture post-apply state.
	// For direct: Finalize flushes the WAL to disk and returns the state;
	// called even if Apply failed so partial progress is saved.
	// For terraform: ParseResourcesState reads the file written by terraform.Apply.
	var (
		state statemgmt.ExportedResourcesMap
		err   error
	)
	if targetEngine.IsDirect() {
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
	statemgmt.PushResourcesState(ctx, b, targetEngine)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(state),
		metadata.Compute(),
		metadata.Upload(),
		statemgmt.UploadStateForYamlSync(targetEngine),
	)

	if !logdiag.HasError(ctx) {
		cmdio.LogString(ctx, "Deployment complete!")
	}

	// Once the deploy is complete, dry-run the migration to the direct engine in
	// memory and record the outcome in telemetry. It writes nothing and never
	// fails the deploy.
	if !targetEngine.IsDirect() && !logdiag.HasError(ctx) {
		statemgmt.CheckDirectMigration(ctx, b)
	}
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
func Deploy(ctx context.Context, b *bundle.Bundle, outputHandler sync.OutputHandler, engine engine.EngineType, libs map[string][]libraries.LocationToUpdate, plan *deployplan.Plan) {
	log.Info(ctx, "Phase: deploy")

	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	bundle.ApplySeqContext(ctx, b,
		scripts.Execute(config.ScriptPreDeploy),
		lock.Acquire(),
	)

	if logdiag.HasError(ctx) {
		// lock is not acquired here
		return
	}

	// lock is acquired here
	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDeploy))
	}()

	immutable := b.IsImmutableFolder()
	if immutable && !engine.IsDirect() {
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

	planFromFile := plan != nil
	if plan == nil {
		// State is already open for read by process.go (for direct engine)
		plan = RunPlan(ctx, b, engine)
	}

	// Stop before opening the WAL for write if planning failed. UpgradeToWrite
	// writes a WAL header that only deployCore's Finalize commits or discards;
	// returning past it without finalizing leaves a header-only WAL behind.
	if logdiag.HasError(ctx) {
		return
	}

	if engine.IsDirect() {
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

	haveApproval, err := approvalForDeploy(ctx, b, plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if haveApproval {
		deployCore(ctx, b, plan, engine)
	} else {
		cmdio.LogString(ctx, "Deployment cancelled!")
		return
	}

	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, scripts.Execute(config.ScriptPostDeploy))
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
