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
	trailingGap  bool   // print an empty line after the block (destroy only)
}

// logApprovalGroups filters actions per group and prints non-empty groups.
// Returns the total number of matched actions across all groups.
func logApprovalGroups(ctx context.Context, actions []deployplan.Action, groups []approvalGroup, types ...deployplan.ActionType) int {
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
		if g.trailingGap {
			cmdio.LogString(ctx, "")
		}
	}
	return total
}
