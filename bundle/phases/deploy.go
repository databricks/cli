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

// resourcesSafeToDestroy lists resource types that can be safely deleted or
// recreated without a confirmation prompt. All other resource types require
// approval because their deletion may cause non-recoverable data loss.
//
// Rubric: a resource is safe if it holds only ephemeral state or configuration
// that is fully recoverable by redeploying the bundle.
//
// Resources NOT on this list (and thus requiring a warning):
//   - schemas: contains tables, views, functions; force-delete cascades to all data.
//   - volumes: managed volumes have files deleted from cloud within 30 days.
//   - pipelines: deletion currently cascades to managed Streaming Tables and Materialized Views.
//   - dashboards: non-reproducible URL/ID; UI-developed content may not be in bundle config.
//   - catalogs: top-level UC container; force-delete cascades to all schemas/tables/data.
//   - secret_scopes: secrets are added out-of-band and destroyed on scope deletion.
//   - database_instances: purge deletes all Postgres data permanently.
//   - database_catalogs: destroys associated Postgres database and all tables/data.
//   - postgres_projects: cascades to delete all branches, databases, and endpoints.
//   - postgres_branches: contains forked database data that is permanently lost.
//   - models: deletion cascades to all model versions and artifacts.
//   - registered_models: deletion cascades to all UC model versions.
//   - experiments: runs, metrics, parameters, and artifacts are lost.
//   - quality_monitors: drift/profile metrics tables may be lost or orphaned.
//   - alerts: purge permanently destroys evaluation and notification history.
var resourcesSafeToDestroy = map[string]bool{
	// Jobs: run history persists independently of the job definition.
	"jobs": true,
	// Model serving endpoints: stateless config; inference tables live in UC independently.
	"model_serving_endpoints": true,
	// Clusters: pure ephemeral compute; all config is in the bundle.
	"clusters": true,
	// Apps: stateless; all config and code deployed from bundle.
	"apps": true,
	// SQL warehouses: compute endpoint; query history stored separately.
	"sql_warehouses": true,
	// External locations: metadata pointer only; underlying cloud storage is not deleted.
	"external_locations": true,
	// Synced database tables: PurgeData=false preserves synced data; source always preserved.
	"synced_database_tables": true,
	// Postgres endpoints: stateless connection config; data lives in branch/project.
	"postgres_endpoints": true,
}

func approvalForDeploy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (bool, error) {
	actions := plan.GetActions()

	err := checkForPreventDestroy(b, actions)
	if err != nil {
		return false, err
	}

	types := []deployplan.ActionType{deployplan.Recreate, deployplan.Delete}

	// Collect destructive actions for resource types that are NOT safe to destroy.
	var destructiveActions []deployplan.Action
	for _, action := range actions {
		actionGroup := config.GetResourceTypeFromKey(action.ResourceKey)
		if resourcesSafeToDestroy[actionGroup] {
			continue
		}
		for _, t := range types {
			if action.ActionType == t {
				destructiveActions = append(destructiveActions, action)
				break
			}
		}
	}

	// We don't need to display any prompts in this case.
	if len(destructiveActions) == 0 {
		return true, nil
	}

	cmdio.LogString(ctx, deleteOrRecreateResourceMessage)
	for _, action := range destructiveActions {
		if action.IsChildResource() {
			continue
		}
		cmdio.Log(ctx, action)
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
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(), plan, direct.MigrateMode(false))
	} else {
		bundle.ApplyContext(ctx, b, terraform.Apply())
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
		// Initialize DeploymentBundle for applying the loaded plan
		_, localPath := b.StateFilenameDirect(ctx)
		err := b.DeploymentBundle.InitForApply(ctx, b.WorkspaceClient(), localPath, plan)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		plan = RunPlan(ctx, b, engine)
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

	logDeployTelemetry(ctx, b)
	bundle.ApplyContext(ctx, b, scripts.Execute(config.ScriptPostDeploy))
}

func RunPlan(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) *deployplan.Plan {
	if engine.IsDirect() {
		_, localPath := b.StateFilenameDirect(ctx)
		plan, err := b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(), &b.Config, localPath)
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
