package phases

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/apps"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
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
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
)

func getActions(ctx context.Context, b *bundle.Bundle) ([]deployplan.Action, error) {
	if b.DirectDeployment {
		err := b.OpenStateFile(ctx)
		if err != nil {
			return nil, err
		}
		err = b.DeploymentBundle.CalculatePlanForDeploy(ctx, b.WorkspaceClient(), &b.Config)
		if err != nil {
			return nil, err
		}
		return b.DeploymentBundle.GetActions(ctx), nil
	} else {
		tf := b.Terraform
		if tf == nil {
			return nil, errors.New("terraform not initialized")
		}
		actions, err := terraform.ShowPlanFile(ctx, tf, b.TerraformPlanPath)
		return actions, err
	}
}

func approvalForDeploy(ctx context.Context, b *bundle.Bundle) (bool, error) {
	actions, err := getActions(ctx, b)
	if err != nil {
		return false, err
	}

	types := []deployplan.ActionType{deployplan.ActionTypeRecreate, deployplan.ActionTypeDelete}
	schemaActions := deployplan.FilterGroup(actions, "schemas", types...)
	dltActions := deployplan.FilterGroup(actions, "pipelines", types...)
	volumeActions := deployplan.FilterGroup(actions, "volumes", types...)
	dashboardActions := deployplan.FilterGroup(actions, "dashboards", types...)

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

func deployCore(ctx context.Context, b *bundle.Bundle) {
	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	cmdio.LogString(ctx, "Deploying resources...")

	if b.DirectDeployment {
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(), &b.Config)
	} else {
		bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	// Even if deployment failed, there might be updates in states that we need to upload
	bundle.ApplyContext(ctx, b,
		statemgmt.StatePush(),
	)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(),

		// TODO: this does terraform specific transformation.
		apps.InterpolateVariables(),

		// TODO: this should either be part of app resource or separate AppConfig resource that depends on main resource.
		apps.UploadConfig(),

		metadata.Compute(),
		metadata.Upload(),
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
func Deploy(ctx context.Context, b *bundle.Bundle, outputHandler sync.OutputHandler) {
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

	libs := deployPrepare(ctx, b)
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
	)

	if logdiag.HasError(ctx) {
		return
	}

	if !b.DirectDeployment {
		bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Plan(terraform.PlanGoal("deploy")),
		)
	}

	if logdiag.HasError(ctx) {
		return
	}

	haveApproval, err := approvalForDeploy(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if haveApproval {
		deployCore(ctx, b)
	} else {
		cmdio.LogString(ctx, "Deployment cancelled!")
	}

	if logdiag.HasError(ctx) {
		return
	}

	logDeployTelemetry(ctx, b)
	bundle.ApplyContext(ctx, b, scripts.Execute(config.ScriptPostDeploy))
}

func Diff(ctx context.Context, b *bundle.Bundle) []deployplan.Action {
	deployPrepare(ctx, b)
	if logdiag.HasError(ctx) {
		return nil
	}

	if !b.DirectDeployment {
		bundle.ApplySeqContext(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.Plan(terraform.PlanGoal("deploy")),
		)
	}

	if logdiag.HasError(ctx) {
		return nil
	}

	actions, err := getActions(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
	}

	return actions
}

// If there are more than 1 thousand of a resource type, do not
// include more resources.
// Since we have a timeout of 3 seconds, we cap the maximum number of IDs
// we send in a single request to have reliable telemetry.
const ResourceIdLimit = 1000
