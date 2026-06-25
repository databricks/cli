package phases

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/metadata"
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

func deployCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, targetEngine engine.EngineType) error {
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
		applyErr := b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan)
		var finalizeErr error
		state, finalizeErr = b.DeploymentBundle.StateDB.Finalize(ctx)
		// Capture the finalized state for deploy telemetry. It carries each
		// resource's state-size in bytes (from the WAL replay Finalize just
		// did), so telemetry needs no extra read or parse of the state file.
		b.Metrics.ResourceState = state
		err = errors.Join(applyErr, finalizeErr)
	} else {
		applyErr := bundle.ApplyContext(ctx, b, terraform.Apply())
		var parseErr error
		state, parseErr = terraform.ParseResourcesState(ctx, b)
		err = errors.Join(applyErr, parseErr)
	}

	// Flush the deploy error now, before the (potentially slow) state push and
	// metadata upload below, so the user sees the failure before that work runs.
	err = logdiag.FlushError(ctx, err)

	// Even if deployment failed, there might be updates in states that we need to upload
	if pushErr := statemgmt.PushResourcesState(ctx, b, targetEngine); pushErr != nil {
		return errors.Join(err, logdiag.FlushError(ctx, pushErr))
	}

	if seqErr := bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(state),
		metadata.Compute(),
		metadata.Upload(),
		statemgmt.UploadStateForYamlSync(targetEngine),
	); seqErr != nil {
		return errors.Join(err, logdiag.FlushError(ctx, seqErr))
	}

	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Deployment complete!")
	return nil
}

// uploadLibraries uploads libraries to the workspace.
// It also cleans up the artifacts directory and transforms wheel tasks.
// It is called by only "bundle deploy".
func uploadLibraries(ctx context.Context, b *bundle.Bundle, libs map[string][]libraries.LocationToUpdate) error {
	return bundle.ApplySeqContext(ctx, b,
		artifacts.CleanUp(),
		libraries.Upload(libs),
	)
}

// The deploy phase deploys artifacts and resources.
// If readPlanPath is provided, the plan is loaded from that file instead of being calculated.
func Deploy(ctx context.Context, b *bundle.Bundle, outputHandler sync.OutputHandler, engine engine.EngineType, libs map[string][]libraries.LocationToUpdate, plan *deployplan.Plan) (err error) {
	log.Info(ctx, "Phase: deploy")

	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	if seqErr := bundle.ApplySeqContext(ctx, b,
		scripts.Execute(config.ScriptPreDeploy),
		lock.Acquire(),
	); seqErr != nil {
		// lock is not acquired here
		return seqErr
	}

	// lock is acquired here
	defer func() {
		// Flush the deploy error before releasing the lock so the user sees the
		// failure before this final API call runs.
		err = logdiag.FlushError(ctx, err)
		if releaseErr := bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDeploy)); releaseErr != nil && err == nil {
			err = logdiag.FlushError(ctx, releaseErr)
		}
	}()

	if uploadErr := uploadLibraries(ctx, b, libs); uploadErr != nil {
		return uploadErr
	}

	if seqErr := bundle.ApplySeqContext(ctx, b,
		files.Upload(outputHandler),
		deploy.StateUpdate(),
		deploy.StatePush(),
		permissions.ApplyWorkspaceRootPermissions(),
		metrics.TrackUsedCompute(),
		deploy.ResourcePathMkdir(),
	); seqErr != nil {
		return seqErr
	}

	planFromFile := plan != nil
	if plan == nil {
		// State is already open for read by process.go (for direct engine).
		// Stop before opening the WAL for write if planning failed. UpgradeToWrite
		// writes a WAL header that only deployCore's Finalize commits or discards;
		// returning past it without finalizing leaves a header-only WAL behind.
		var planErr error
		plan, planErr = RunPlan(ctx, b, engine)
		if planErr != nil {
			return planErr
		}
	}

	if engine.IsDirect() {
		// Upgrade from read (opened by process.go) to write mode
		if upgradeErr := b.DeploymentBundle.StateDB.UpgradeToWrite(); upgradeErr != nil {
			return upgradeErr
		}
	}

	if planFromFile {
		// Initialize DeploymentBundle for applying the loaded plan
		if initErr := b.DeploymentBundle.InitForApply(ctx, b.WorkspaceClient(ctx), plan); initErr != nil {
			return initErr
		}
	}

	haveApproval, approvalErr := approvalForDeploy(ctx, b, plan)
	if approvalErr != nil {
		return approvalErr
	}
	if !haveApproval {
		cmdio.LogString(ctx, "Deployment cancelled!")
		return nil
	}

	if coreErr := deployCore(ctx, b, plan, engine); coreErr != nil {
		return coreErr
	}

	return bundle.ApplyContext(ctx, b, scripts.Execute(config.ScriptPostDeploy))
}

func RunPlan(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) (*deployplan.Plan, error) {
	if engine.IsDirect() {
		plan, err := b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), &b.Config)
		if err != nil {
			return nil, err
		}
		if len(b.Select) > 0 {
			plan.FilterToSelected(b.Select)
		}
		return plan, nil
	}

	// b.Select is rejected for the terraform engine in ProcessBundleRet, so it is
	// never set here.

	if err := bundle.ApplySeqContext(ctx, b,
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Plan(terraform.PlanGoal("deploy")),
	); err != nil {
		return nil, err
	}

	tf := b.Terraform
	if tf == nil {
		return nil, errors.New("terraform not initialized")
	}

	plan, err := terraform.ShowPlanFile(ctx, tf, b.TerraformPlanPath)
	if err != nil {
		return nil, err
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

	return plan, nil
}

// If there are more than 1 thousand of a resource type, do not
// include more resources.
// Since we have a timeout of 3 seconds, we cap the maximum number of IDs
// we send in a single request to have reliable telemetry.
const ResourceIdLimit = 1000
