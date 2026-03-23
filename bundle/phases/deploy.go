package phases

import (
	"context"
	"errors"
	"strings"

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

// resourceDestroyWarning returns the warning message for a given resource type,
// or an empty string if the resource is safe to delete/recreate.
// For child resources (e.g. permissions, grants), the parent type is used.
func resourceDestroyWarning(action deployplan.Action) string {
	resourceType := config.GetResourceTypeFromKey(action.ResourceKey)
	if base, _, ok := strings.Cut(resourceType, "."); ok {
		resourceType = base
	}
	return resourceDestroyMessage[resourceType]
}

func approvalForDeploy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (bool, error) {
	actions := plan.GetActions()

	err := checkForPreventDestroy(b, actions)
	if err != nil {
		return false, err
	}

	destructiveActions := collectDestructiveActions(actions)

	// We don't need to display any prompts in this case.
	if len(destructiveActions) == 0 {
		return true, nil
	}

	logDestructiveActions(ctx, destructiveActions)

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

// collectDestructiveActions returns delete/recreate actions for resource types
// that are not safe to destroy. Child resources (permissions, grants) are excluded.
func collectDestructiveActions(actions []deployplan.Action) []deployplan.Action {
	var result []deployplan.Action
	for _, action := range actions {
		if action.IsChildResource() {
			continue
		}
		if action.ActionType != deployplan.Recreate && action.ActionType != deployplan.Delete {
			continue
		}
		if resourceDestroyWarning(action) == "" {
			continue
		}
		result = append(result, action)
	}
	return result
}

// logDestructiveActions prints grouped warning messages for destructive actions.
// Actions are grouped by their warning message so that each unique message is
// printed once, followed by the affected resources.
func logDestructiveActions(ctx context.Context, actions []deployplan.Action) {
	// Use a slice to preserve insertion order of messages.
	var messages []string
	groups := map[string][]deployplan.Action{}
	for _, action := range actions {
		msg := resourceDestroyWarning(action)
		if _, seen := groups[msg]; !seen {
			messages = append(messages, msg)
		}
		groups[msg] = append(groups[msg], action)
	}

	for _, msg := range messages {
		cmdio.LogString(ctx, "\n"+msg)
		for _, action := range groups[msg] {
			cmdio.Log(ctx, action)
		}
	}
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
