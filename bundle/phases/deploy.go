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
	"github.com/databricks/cli/bundle/resourcesnapshot"
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

	types := []deployplan.ActionType{deployplan.ActionTypeRecreate, deployplan.ActionTypeDelete}
	schemaActions := filterGroup(actions, "schemas", types...)
	dltActions := filterGroup(actions, "pipelines", types...)
	volumeActions := filterGroup(actions, "volumes", types...)
	dashboardActions := filterGroup(actions, "dashboards", types...)

	// We don't need to display any prompts in this case.
	if len(schemaActions) == 0 && len(dltActions) == 0 && len(volumeActions) == 0 && len(dashboardActions) == 0 {
		return true, nil
	}

	// One or more UC schema resources will be deleted or recreated.
	if len(schemaActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateSchemaMessage)
		for _, action := range schemaActions {
			cmdio.Log(ctx, action)
		}
	}

	// One or more DLT pipelines is being recreated.
	if len(dltActions) != 0 {
		cmdio.LogString(ctx, deleteOrRecreateDltMessage)
		for _, action := range dltActions {
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
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(), &b.Config, plan)
	} else {
		bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	// Even if deployment failed, there might be updates in states that we need to upload
	bundle.ApplyContext(ctx, b,
		statemgmt.StatePush(targetEngine),
	)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(targetEngine),
		metadata.Compute(),
		metadata.Upload(),
		resourcesnapshot.Save(),
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
func Deploy(ctx context.Context, b *bundle.Bundle, outputHandler sync.OutputHandler, engine engine.EngineType) {
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

	libs := DeployPrepare(ctx, b, false, engine)
	if logdiag.HasError(ctx) {
		return
	}

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

	plan := RunPlan(ctx, b, engine)
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
					Action: deployplan.ActionTypeSkip.String(),
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
