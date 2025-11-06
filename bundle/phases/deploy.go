package phases

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/databricks/cli/bundle"
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

func deployCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, directDeployment bool) {
	// Core mutators that CRUD resources and modify deployment state. These
	// mutators need informed consent if they are potentially destructive.
	cmdio.LogString(ctx, "Deploying resources...")

	if directDeployment {
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(), &b.Config, plan)
	} else {
		bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	// Even if deployment failed, there might be updates in states that we need to upload
	// Use context.WithoutCancel to ensure state push completes even if context was cancelled
	// (e.g., due to signal interruption). We want to save the current state before exiting.
	statePushCtx := context.WithoutCancel(ctx)
	bundle.ApplyContext(statePushCtx, b,
		statemgmt.StatePush(directDeployment),
	)
	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(directDeployment),
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

// registerGracefulCleanup sets up signal handlers to release the lock
// before the process terminates. Returns a new context that will be cancelled
// when a signal is received, and a cleanup function for the exit path.
//
// This follows idiomatic Go patterns for graceful shutdown:
// 1. Use context cancellation to signal shutdown to the main routine
// 2. Use a done channel to wait for the main routine to complete
// 3. Only exit after confirming the main routine has terminated
//
// Catches SIGINT (Ctrl+C), SIGTERM, SIGHUP, and SIGQUIT.
// Note: SIGKILL and SIGSTOP cannot be caught - the kernel terminates the process directly.
func registerGracefulCleanup(ctx context.Context, b *bundle.Bundle) (context.Context, func()) {
	// Create a cancellable context to propagate cancellation to the main routine
	ctx, cancel := context.WithCancel(ctx)

	// Channel to signal that the main + cleanup routine has completed
	cleanupDone := make(chan struct{})

	// Channel to signal that a signal was received and handled
	signalReceived := make(chan struct{})

	// Channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	signalHandler := func() {
		// Wait for a signal to be received.
		sig := <-sigChan

		// Stop listening for more signals. This allows for multiple interrupts
		// to cause the program to force exit.
		signal.Stop(sigChan)

		// Signal that we received an interrupt
		close(signalReceived)

		cmdio.LogString(ctx, "Operation interrupted. Gracefully shutting down...")

		// Cancel the context to signal the main routine to stop
		cancel()

		// Wait for the main routine to complete before releasing the lock
		// This ensures we don't exit while operations are still in progress
		<-cleanupDone

		// Release the lock using a context without cancellation to avoid cancellation issues
		// We use context.WithoutCancel to preserve context values (like user agent)
		// but remove the cancellation signal so the lock release can complete
		releaseCtx := context.WithoutCancel(ctx)
		bundle.ApplyContext(releaseCtx, b, lock.Release())

		// Calculate exit code (128 + signal number)
		exitCode := 128
		if s, ok := sig.(syscall.Signal); ok {
			exitCode += int(s)
		}

		// Exit with the appropriate signal exit code
		os.Exit(exitCode)
	}

	// Start goroutine to handle signals
	go signalHandler()

	// Return cleanup function for the exit path
	// This should be called via defer to ensure it runs even if there's a panic
	cleanup := func() {
		// Stop listening for signals
		signal.Stop(sigChan)

		// Release the lock (idempotent)
		// Use context.WithoutCancel to preserve context values but remove cancellation
		releaseCtx := context.WithoutCancel(ctx)
		bundle.ApplyContext(releaseCtx, b, lock.Release())

		// Signal that the main routine has completed.
		// Once the signal is recieved,
		// This must be done AFTER all cleanup is complete
		close(cleanupDone)

		// If a signal was received, wait indefinitely for the signal handler to exit
		// This prevents the main function from returning and exiting with a different code
		// If no signal was received, signalReceived will never be closed, so we just return
		select {
		case <-signalReceived:
			// Signal was received, wait forever for os.Exit() in signal handler
			select {}
		default:
			// No signal received, proceed with normal exit
		}
	}

	return ctx, cleanup
}

// The deploy phase deploys artifacts and resources.
func Deploy(ctx context.Context, b *bundle.Bundle, outputHandler sync.OutputHandler, directDeployment bool) {
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

	// lock is acquired here - set up signal handlers and defer cleanup
	ctx, cleanup := registerGracefulCleanup(ctx, b)
	defer cleanup()

	libs := deployPrepare(ctx, b, false, directDeployment)
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

	plan := planWithoutPrepare(ctx, b, directDeployment)
	if logdiag.HasError(ctx) {
		return
	}

	haveApproval, err := approvalForDeploy(ctx, b, plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if haveApproval {
		deployCore(ctx, b, plan, directDeployment)
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

// planWithoutPrepare builds a deployment plan without running deployPrepare.
// This is used when deployPrepare has already been called.
func planWithoutPrepare(ctx context.Context, b *bundle.Bundle, directDeployment bool) *deployplan.Plan {
	if directDeployment {
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

func Plan(ctx context.Context, b *bundle.Bundle, directDeployment bool) *deployplan.Plan {
	deployPrepare(ctx, b, true, directDeployment)
	if logdiag.HasError(ctx) {
		return nil
	}

	return planWithoutPrepare(ctx, b, directDeployment)
}

// If there are more than 1 thousand of a resource type, do not
// include more resources.
// Since we have a timeout of 3 seconds, we cap the maximum number of IDs
// we send in a single request to have reliable telemetry.
const ResourceIdLimit = 1000
