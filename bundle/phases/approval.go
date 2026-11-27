package phases

import (
	"context"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
)

// approvalGroup describes one resource type that needs explicit user consent
// before a destructive action is applied.
type approvalGroup struct {
	group        string // matches config.GetResourceTypeFromKey, e.g. "schemas"
	message      string // banner shown above the action list
	skipChildren bool   // skip actions where IsChildResource() is true
}

// approvalEntry holds the deploy and destroy messages of one resource type.
type approvalEntry struct {
	group              string
	deployMessage      string
	deploySkipChildren bool
	// destroyMessage is empty when destroy prints no message from the table for the group.
	destroyMessage string
}

// approvalTable lists the resource types whose deletion or recreation needs consent.
// Deploy asks for approval only when the plan deletes or recreates a resource in one of
// these groups. Destroy asks whatever it deletes, and first prints the message of each
// group it deletes resources from. Messages print in table order.
var approvalTable = []approvalEntry{
	{group: "schemas", deployMessage: deleteOrRecreateSchemaMessage, deploySkipChildren: true, destroyMessage: deleteSchemaMessage},
	// Destroy prints the pipeline messages in logPipelineDeleteApproval, which picks one
	// per pipeline from its cascade_on_destroy setting.
	{group: "pipelines", deployMessage: deleteOrRecreatePipelineMessage},
	{group: "volumes", deployMessage: deleteOrRecreateVolumeMessage, destroyMessage: deleteVolumeMessage},
	// No destroy message: the deploy message warns that a recreated dashboard gets a new
	// ID and URL, and destroy recreates nothing.
	{group: "dashboards", deployMessage: deleteOrRecreateDashboardMessage},
	{group: "database_instances", deployMessage: deleteOrRecreateDatabaseInstanceMessage, destroyMessage: deleteDatabaseInstanceMessage},
	{group: "synced_database_tables", deployMessage: deleteOrRecreateSyncedDatabaseTableMessage, destroyMessage: deleteSyncedDatabaseTableMessage},
	{group: "postgres_projects", deployMessage: deleteOrRecreatePostgresProjectMessage, destroyMessage: deletePostgresProjectMessage},
	{group: "postgres_branches", deployMessage: deleteOrRecreatePostgresBranchMessage, destroyMessage: deletePostgresBranchMessage},
	{group: "postgres_databases", deployMessage: deleteOrRecreatePostgresDatabaseMessage, destroyMessage: deletePostgresDatabaseMessage},
	{group: "vector_search_indexes", deployMessage: deleteOrRecreateVectorSearchIndexMessage, destroyMessage: deleteVectorSearchIndexMessage},
	{group: "genie_spaces", deployMessage: deleteOrRecreateGenieSpaceMessage, destroyMessage: deleteGenieSpaceMessage},
	// No destroy message: the deploy message warns that a recreated set recomputes every
	// member, and destroy recreates nothing.
	{group: "materialization_sets", deployMessage: deleteOrRecreateMaterializationSetMessage},
	{group: "features", deployMessage: deleteOrRecreateFeatureMessage, destroyMessage: deleteFeatureMessage},
	{group: "feature_specs", deployMessage: deleteOrRecreateFeatureSpecMessage, destroyMessage: deleteFeatureSpecMessage},
}

// deployApprovalGroups returns the groups deploy asks for approval on.
func deployApprovalGroups() []approvalGroup {
	groups := make([]approvalGroup, 0, len(approvalTable))
	for _, e := range approvalTable {
		groups = append(groups, approvalGroup{group: e.group, message: e.deployMessage, skipChildren: e.deploySkipChildren})
	}
	return groups
}

// destroyApprovalGroups returns the groups whose destroy message comes from the table.
func destroyApprovalGroups() []approvalGroup {
	var groups []approvalGroup
	for _, e := range approvalTable {
		if e.destroyMessage == "" {
			continue
		}
		groups = append(groups, approvalGroup{group: e.group, message: e.destroyMessage})
	}
	return groups
}

// logApprovalGroups filters actions per group and prints non-empty groups.
// If trailingNewline is true, an empty line is printed after each non-empty group.
// Returns the total number of matched actions across all groups.
func logApprovalGroups(ctx context.Context, actions []deployplan.Action, groups []approvalGroup, trailingNewline bool, types ...deployplan.ActionType) int {
	total := 0
	for _, g := range groups {
		matched := filterGroup(actions, g.group, types...)
		if len(matched) == 0 {
			continue
		}
		total += len(matched)
		cmdio.LogString(ctx, g.message)
		for _, a := range matched {
			if g.skipChildren && a.IsChildResource() {
				continue
			}
			cmdio.Log(ctx, a)
		}
		if trailingNewline {
			cmdio.LogString(ctx, "")
		}
	}
	return total
}
