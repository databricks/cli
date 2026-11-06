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

// approvalEntry is one resource type in approvalTable, the single list that
// deploy and destroy derive their prompts from.
type approvalEntry struct {
	group              string
	deployMessage      string
	deploySkipChildren bool
	// destroyMessage is nil when destroy has no prompt of its own for the group.
	destroyMessage *string
}

// approvalTable is in deploy's order. Destroy's groups keep the same relative order.
var approvalTable = []approvalEntry{
	{group: "schemas", deployMessage: deleteOrRecreateSchemaMessage, deploySkipChildren: true, destroyMessage: new(deleteSchemaMessage)},
	// Destroy prompts for pipelines in logPipelineDeleteApproval, which picks the message
	// per pipeline from its cascade_on_destroy setting.
	{group: "pipelines", deployMessage: deleteOrRecreatePipelineMessage},
	{group: "volumes", deployMessage: deleteOrRecreateVolumeMessage, destroyMessage: new(deleteVolumeMessage)},
	// Deploy only.
	{group: "dashboards", deployMessage: deleteOrRecreateDashboardMessage},
	{group: "database_instances", deployMessage: deleteOrRecreateDatabaseInstanceMessage, destroyMessage: new(deleteDatabaseInstanceMessage)},
	{group: "synced_database_tables", deployMessage: deleteOrRecreateSyncedDatabaseTableMessage, destroyMessage: new(deleteSyncedDatabaseTableMessage)},
	{group: "postgres_projects", deployMessage: deleteOrRecreatePostgresProjectMessage, destroyMessage: new(deletePostgresProjectMessage)},
	{group: "postgres_branches", deployMessage: deleteOrRecreatePostgresBranchMessage, destroyMessage: new(deletePostgresBranchMessage)},
	{group: "postgres_databases", deployMessage: deleteOrRecreatePostgresDatabaseMessage, destroyMessage: new(deletePostgresDatabaseMessage)},
	{group: "vector_search_indexes", deployMessage: deleteOrRecreateVectorSearchIndexMessage, destroyMessage: new(deleteVectorSearchIndexMessage)},
	{group: "genie_spaces", deployMessage: deleteOrRecreateGenieSpaceMessage, destroyMessage: new(deleteGenieSpaceMessage)},
	// Deploy only.
	{group: "materialization_sets", deployMessage: deleteOrRecreateMaterializationSetMessage},
	{group: "features", deployMessage: deleteOrRecreateFeatureMessage, destroyMessage: new(deleteFeatureMessage)},
}

var deployApprovalGroups = deployGroups()

var destroyApprovalGroups = destroyGroups()

// deployGroups returns the groups deploy prompts for, with the skipChildren setting for schemas.
func deployGroups() []approvalGroup {
	groups := make([]approvalGroup, 0, len(approvalTable))
	for _, e := range approvalTable {
		groups = append(groups, approvalGroup{group: e.group, message: e.deployMessage, skipChildren: e.deploySkipChildren})
	}
	return groups
}

// destroyGroups returns the groups destroy prompts for through logApprovalGroups.
func destroyGroups() []approvalGroup {
	var groups []approvalGroup
	for _, e := range approvalTable {
		if e.destroyMessage == nil {
			continue
		}
		groups = append(groups, approvalGroup{group: e.group, message: *e.destroyMessage})
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
