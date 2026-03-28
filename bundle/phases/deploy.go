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
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
)

func approvalForDeploy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (bool, error) {
	actions := plan.GetActions()

	err := checkForPreventDestroy(b, actions)
	if err != nil {
		return false, err
	}

	types := []deployplan.ActionType{deployplan.Recreate, deployplan.Delete}
	schemaActions := filterGroup(actions, "schemas", types...)
	pipelineActions := filterGroup(actions, "pipelines", types...)
	volumeActions := filterGroup(actions, "volumes", types...)
	dashboardActions := filterGroup(actions, "dashboards", types...)
	databaseInstanceActions := filterGroup(actions, "database_instances", types...)
	syncedDatabaseTableActions := filterGroup(actions, "synced_database_tables", types...)
	postgresProjectActions := filterGroup(actions, "postgres_projects", types...)
	postgresBranchActions := filterGroup(actions, "postgres_branches", types...)

	// We don't need to display any prompts in this case.
	if len(schemaActions) == 0 && len(pipelineActions) == 0 && len(volumeActions) == 0 && len(dashboardActions) == 0 &&
		len(databaseInstanceActions) == 0 && len(syncedDatabaseTableActions) == 0 &&
		len(postgresProjectActions) == 0 && len(postgresBranchActions) == 0 {
		return true, nil
	}

	// One or more UC schema resources will be deleted or recreated.
	if len(schemaActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateSchemaMessage)
		for _, action := range schemaActions {
			if action.IsChildResource() {
				continue
			}
			cmdio.Log(ctx, action)
		}
	}

	// One or more pipelines is being recreated.
	if len(pipelineActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreatePipelineMessage)
		for _, action := range pipelineActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more volumes is being recreated.
	if len(volumeActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateVolumeMessage)
		for _, action := range volumeActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more dashboards is being recreated.
	if len(dashboardActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateDashboardMessage)
		for _, action := range dashboardActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more database instances is being deleted or recreated.
	if len(databaseInstanceActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateDatabaseInstanceMessage)
		for _, action := range databaseInstanceActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more synced database tables is being deleted or recreated.
	if len(syncedDatabaseTableActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateSyncedDatabaseTableMessage)
		for _, action := range syncedDatabaseTableActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more Lakebase projects is being deleted or recreated.
	if len(postgresProjectActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreatePostgresProjectMessage)
		for _, action := range postgresProjectActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more Lakebase branches is being deleted or recreated.
	if len(postgresBranchActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreatePostgresBranchMessage)
		for _, action := range postgresBranchActions {
			cmdio.Log(ctx, action)
		}
	}

	if b.AutoApprove {
		return true, nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		return false, errors.New("the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
	}

	cmdio.LogString(ctx, "")
	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return false, err
	}

	return approved, nil
}

func deployCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, targetEngine engine.EngineType) {
	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	cmdio.LogString(ctx, "Deploying resources...")

	if targetEngine.IsDirect() {
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan, direct.MigrateMode(false))
		// Finalize state: write to disk even if deploy failed, so partial progress is saved.
		// Skip for empty plans to avoid creating a state file when nothing was deployed.
		if len(plan.Plan) > 0 {
			if err := b.DeploymentBundle.StateDB.Finalize(); err != nil {
				logdiag.LogError(ctx, err)
			}
		}
	} else {
		bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	// Close state to replay WAL into state file, then reopen for read.
	// PushResourcesState needs the file on disk, Load needs the state in memory.
	if targetEngine.IsDirect() {
		if err := b.DeploymentBundle.StateDB.Close(ctx); err != nil {
			logdiag.LogError(ctx, err)
		}
		_, localPath := b.StateFilenameDirect(ctx)
		if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false)); err != nil {
			logdiag.LogError(ctx, err)
		}
	}

	// Even if deployment failed, there might be updates in states that we need to upload
	statemgmt.PushResourcesState(ctx, b, targetEngine)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(targetEngine),
		metadata.Compute(),
		metadata.Upload(),
		statemgmt.UploadStateForYamlSync(targetEngine),
	)

	if !logdiag.HasError(ctx) {
		cmdio.LogString(ctx, "Deployment complete!")
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

	uploadLibraries(ctx, b, libs)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		files.Upload(outputHandler),
		deploy.StateUpdate(),
		deploy.StatePush(),
		permissions.ApplyWorkspaceRootPermissions(),
		metrics.TrackUsedCompute(),
		deploy.ResourcePathMkdir(),
	)

	if logdiag.HasError(ctx) {
		return
	}

	if plan != nil {
		if engine.IsDirect() {
			// Upgrade from read (opened by process.go) to write mode
			if err := b.DeploymentBundle.StateDB.UpgradeToWrite(); err != nil {
				logdiag.LogError(ctx, err)
				return
			}
			defer func() {
				if err := b.DeploymentBundle.StateDB.Close(ctx); err != nil {
					logdiag.LogError(ctx, err)
				}
			}()
		}
		// Initialize DeploymentBundle for applying the loaded plan
		err := b.DeploymentBundle.InitForApply(ctx, b.WorkspaceClient(ctx), plan)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		// State is already open for read by process.go (for direct engine)
		plan = RunPlan(ctx, b, engine)
		if engine.IsDirect() {
			// Upgrade from read to write mode (Apply needs write access)
			if err := b.DeploymentBundle.StateDB.UpgradeToWrite(); err != nil {
				logdiag.LogError(ctx, err)
				return
			}
			defer func() {
				if err := b.DeploymentBundle.StateDB.Close(ctx); err != nil {
					logdiag.LogError(ctx, err)
				}
			}()
		}
	}

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
		return plan
	}

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
