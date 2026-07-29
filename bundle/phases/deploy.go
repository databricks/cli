package phases

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/workspaceurls"
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

func deployCore(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, stateEngine engine.EngineType, requestedEngine engine.EngineSetting) {
	// Apply resources and capture post-apply state.
	// For direct: Finalize flushes the WAL to disk and returns the state;
	// called even if Apply failed so partial progress is saved.
	// For terraform: ParseResourcesState reads the file written by terraform.Apply.
	var (
		state statemgmt.ExportedResourcesMap
		err   error
	)
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

	// Report what was deployed, mirroring "bundle plan". Printed only on success
	// and after state/metadata have been uploaded, so a state-push failure is not
	// masked by a success summary.
	if !logdiag.HasError(ctx) {
		logDeploySummary(ctx, b, plan)
	}

	// Once the deploy is complete, dry-run the migration to the direct engine
	// and record the outcome in telemetry. If the user has opted in to the
	// direct engine (via bundle.engine or DATABRICKS_BUNDLE_ENGINE) and the
	// dry-run is clean, the migration is committed; otherwise nothing is
	// written and the deploy is unaffected.
	if !stateEngine.IsDirect() && !logdiag.HasError(ctx) {
		statemgmt.MigrateToDirect(ctx, b, requestedEngine)
	}
}

// logDeploySummary prints the per-resource actions that were applied followed by
// a summary line, mirroring the output of "bundle plan". Each per-resource line
// shows a workspace URL for the resource when one is available. Per-resource
// lines are suppressed by --quiet. The past-tense verb is the short action name
// plus "d" (create→created, delete→deleted, ...).
func logDeploySummary(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) {
	if !b.Quiet {
		logResourceActions(ctx, b, plan)
	}

	counts := plan.CountActions()
	summary := fmt.Sprintf("Deploy: %d created, %d changed, %d deleted, %d unchanged", counts.Create, counts.Change, counts.Delete, counts.Unchanged)
	// Gate on the plan's own NotSelected (not b.Select) so the suffix survives a
	// deploy from a --plan file, where --select was applied at plan time and
	// b.Select is empty here. NotSelected is only ever set by FilterToSelected.
	if plan.NotSelected > 0 {
		summary += fmt.Sprintf(", %d not selected", plan.NotSelected)
	}
	cmdio.LogString(ctx, summary+".")
}

// logResourceActions prints one line per applied action, aligning resource URLs
// into a column. URLs require the workspace ID, whose resolution may cost an API
// call; that only happens here, when the per-resource lines are printed.
func logResourceActions(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) {
	baseURL, err := mutator.WorkspaceBaseURL(ctx, b)
	if err != nil {
		// Fall back to printing the lines without URLs rather than failing an
		// otherwise-successful deploy on a URL-resolution error.
		log.Debugf(ctx, "cannot resolve workspace URL for deploy summary: %v", err)
	}
	withURLs := err == nil

	// Populate the URL field on every live resource in config so it can be read
	// back below. Deleted resources are gone from config, so their URLs are built
	// from the base URL directly.
	if withURLs {
		for _, group := range b.Config.Resources.AllResources() {
			for _, r := range group.Resources {
				r.InitializeURL(baseURL)
			}
		}
	}

	type line struct{ label, url string }
	var lines []line
	labelWidth := 0
	for _, action := range plan.GetActions() {
		if action.ActionType == deployplan.Skip || action.ActionType == deployplan.Undefined {
			continue
		}
		label := action.ActionType.StringShort() + "d " + strings.TrimPrefix(action.ResourceKey, "resources.")
		l := line{label: label}
		if withURLs {
			l.url = resourceURL(b, baseURL, action)
		}
		lines = append(lines, l)
		labelWidth = max(labelWidth, len(label))
	}

	for _, l := range lines {
		if l.url == "" {
			cmdio.LogString(ctx, l.label)
		} else {
			cmdio.LogString(ctx, fmt.Sprintf("%-*s  %s", labelWidth, l.label, l.url))
		}
	}

	if len(lines) > 0 {
		cmdio.LogString(ctx, "")
	}
}

// resourceURL returns the workspace URL for an applied action, or "" when none
// is available (child nodes like grants/permissions, resource types without a
// URL, or a delete whose ID could not be recovered).
func resourceURL(b *bundle.Bundle, baseURL url.URL, action deployplan.Action) string {
	if action.IsChildResource() {
		return ""
	}

	if action.ActionType == deployplan.Delete {
		// A deleted resource is gone from config; build its URL from the ID
		// captured at plan time. The link points at a now-deleted resource, but
		// it identifies what was removed.
		resourceType := config.GetResourceTypeFromKey(action.ResourceKey)
		return workspaceurls.ResourceURL(baseURL, resourceType, action.ID)
	}

	// action.ResourceKey is "resources.<type>.<name>"; Lookup keys on "<type>.<name>".
	key := strings.TrimPrefix(action.ResourceKey, "resources.")
	ref, err := resources.Lookup(b, key)
	if err != nil {
		return ""
	}
	u := ref.Resource.GetURL()
	// Some resources derive their URL from a name that is still an unresolved
	// "${...}" reference at this point (e.g. synced tables keyed by
	// "${resources.catalog.name}...."). Such a URL is not a usable link and its
	// rendering differs by engine, so omit it.
	if strings.Contains(u, "$%7B") || strings.Contains(u, "${") {
		return ""
	}
	return u
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
	defer func() {
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

	haveApproval, err := approvalForDeploy(ctx, b, plan)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	if haveApproval {
		deployCore(ctx, b, plan, stateEngine, requestedEngine)
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
