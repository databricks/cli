package phases

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

func assertRootPathExists(ctx context.Context, b *bundle.Bundle) (bool, error) {
	w := b.WorkspaceClient(ctx)
	_, err := w.Workspace.GetStatusByPath(ctx, b.Config.Workspace.RootPath) //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.

	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
		log.Infof(ctx, "Root path does not exist: %s", b.Config.Workspace.RootPath)
		return false, nil
	}

	return true, err
}

func approvalForDestroy(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (bool, error) {
	deleteActions := plan.GetActions()

	err := checkForPreventDestroy(b, deleteActions)
	if err != nil {
		return false, err
	}

	if len(deleteActions) > 0 {
		cmdio.LogString(ctx, "The following resources will be deleted:")
		for _, a := range deleteActions {
			if a.IsChildResource() {
				continue
			}
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	schemaActions := filterGroup(deleteActions, "schemas", deployplan.Delete)
	pipelineActions := filterGroup(deleteActions, "pipelines", deployplan.Delete)
	volumeActions := filterGroup(deleteActions, "volumes", deployplan.Delete)
	databaseInstanceActions := filterGroup(deleteActions, "database_instances", deployplan.Delete)
	syncedDatabaseTableActions := filterGroup(deleteActions, "synced_database_tables", deployplan.Delete)
	postgresProjectActions := filterGroup(deleteActions, "postgres_projects", deployplan.Delete)
	postgresBranchActions := filterGroup(deleteActions, "postgres_branches", deployplan.Delete)

	if len(schemaActions) > 0 {
		cmdio.LogString(ctx, deleteSchemaMessage)
		for _, a := range schemaActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(pipelineActions) > 0 {
		cmdio.LogString(ctx, deletePipelineMessage)
		for _, a := range pipelineActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(volumeActions) > 0 {
		cmdio.LogString(ctx, deleteVolumeMessage)
		for _, a := range volumeActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(databaseInstanceActions) > 0 {
		cmdio.LogString(ctx, deleteDatabaseInstanceMessage)
		for _, a := range databaseInstanceActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(syncedDatabaseTableActions) > 0 {
		cmdio.LogString(ctx, deleteSyncedDatabaseTableMessage)
		for _, a := range syncedDatabaseTableActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(postgresProjectActions) > 0 {
		cmdio.LogString(ctx, deletePostgresProjectMessage)
		for _, a := range postgresProjectActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	if len(postgresBranchActions) > 0 {
		cmdio.LogString(ctx, deletePostgresBranchMessage)
		for _, a := range postgresBranchActions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}

	cmdio.LogString(ctx, "All files and directories at the following location will be deleted: "+b.Config.Workspace.RootPath)
	cmdio.LogString(ctx, "")

	if b.AutoApprove {
		return true, nil
	}

	approved, err := cmdio.AskYesOrNo(ctx, "Would you like to proceed?")
	if err != nil {
		return false, err
	}

	return approved, nil
}

func destroyCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, engine engine.EngineType) {
	if engine.IsDirect() {
		b.DeploymentBundle.Apply(ctx, b.WorkspaceClient(ctx), plan, direct.MigrateMode(false))
		// Skip Finalize for empty plans to avoid creating a state file when nothing was destroyed.
		if len(plan.Plan) > 0 {
			if err := b.DeploymentBundle.StateDB.Finalize(); err != nil {
				logdiag.LogError(ctx, err)
			}
		}
	} else {
		// Core destructive mutators for destroy. These require informed user consent.
		bundle.ApplyContext(ctx, b, terraform.Apply())
	}

	if logdiag.HasError(ctx) {
		return
	}

	bundle.ApplyContext(ctx, b, files.Delete())

	if !logdiag.HasError(ctx) {
		cmdio.LogString(ctx, "Destroy complete!")
	}
}

// errIfBoundResourcesInState rejects a destroy when any bind block currently
// matches a resource in state: that resource was imported, not created by the
// bundle, and a blanket destroy would delete a pre-existing workspace resource.
func errIfBoundResourcesInState(stateDB *dstate.DeploymentState, bindConfig config.Bind) error {
	var boundInState []string
	bindConfig.ForEach(func(resourceType, resourceName, bindID string) {
		key := "resources." + resourceType + "." + resourceName
		if entry, ok := stateDB.Data.State[key]; ok && entry.ID == bindID {
			boundInState = append(boundInState, key)
		}
	})

	if len(boundInState) == 0 {
		return nil
	}

	slices.Sort(boundInState)
	return fmt.Errorf("cannot destroy with bind blocks that reference resources in the deployment state: %s; remove the bind blocks from the target configuration or run 'bundle deployment unbind' before destroying", strings.Join(boundInState, ", "))
}

// The destroy phase deletes artifacts and resources.
func Destroy(ctx context.Context, b *bundle.Bundle, engine engine.EngineType) {
	log.Info(ctx, "Phase: destroy")

	ok, err := assertRootPathExists(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if !ok {
		cmdio.LogString(ctx, "No active deployment found to destroy!")
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalDestroy))
	}()

	if !engine.IsDirect() {
		bundle.ApplySeqContext(ctx, b,
			// We need to resolve artifact variable (how we do it in build phase)
			// because some of the to-be-destroyed resource might use this variable.
			// Not resolving might lead to terraform "Reference to undeclared resource" error
			mutator.ResolveVariableReferencesWithoutResources("artifacts"),
			mutator.ResolveVariableReferencesOnlyResources("artifacts"),

			terraform.Interpolate(),
			terraform.Write(),
			terraform.Plan(terraform.PlanGoal("destroy")),
		)
	}

	if logdiag.HasError(ctx) {
		return
	}

	var plan *deployplan.Plan
	if engine.IsDirect() {
		// Refuse to destroy when bind blocks point at resources that are currently
		// in state: those are pre-existing workspace resources the user imported,
		// and destroying them would silently delete data the bundle did not create.
		if b.Target != nil && !b.Target.Bind.IsEmpty() {
			if err := errIfBoundResourcesInState(&b.DeploymentBundle.StateDB, b.Target.Bind); err != nil {
				logdiag.LogError(ctx, err)
				return
			}
		}

		plan, err = b.DeploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(ctx), nil)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	} else {
		tf := b.Terraform
		if tf == nil {
			logdiag.LogError(ctx, errors.New("terraform not initialized"))
			return
		}

		plan, err = terraform.ShowPlanFile(ctx, tf, b.TerraformPlanPath)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	hasApproval, err := approvalForDestroy(ctx, b, plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if hasApproval {
		destroyCore(ctx, b, plan, engine)
	} else {
		cmdio.LogString(ctx, "Destroy cancelled!")
	}
}
