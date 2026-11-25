package phases

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
)

// approvalGroup describes one resource type that needs explicit user consent
// before a destructive action is applied.
type approvalGroup struct {
	group        string // matches config.GetResourceTypeFromKey
	message      string // banner shown above the action list
	skipChildren bool   // skip actions where IsChildResource() is true
}

// typeApproval is the [resources.Approval] declared by the resource type under key.
type typeApproval struct {
	key string
	resources.Approval
}

// approvals returns the [resources.Approval] of every resource type, ordered by Order.
func approvals() []typeApproval {
	var res []typeApproval
	for key, d := range config.ResourceTypesImplementing[resources.ApprovalDeclarer]() {
		res = append(res, typeApproval{key: key, Approval: d.Approval()})
	}
	slices.SortStableFunc(res, func(a, b typeApproval) int { return cmp.Compare(a.Order, b.Order) })
	return res
}

// deployApprovalGroups returns the groups deploy asks for approval on.
func deployApprovalGroups() []approvalGroup {
	var groups []approvalGroup
	for _, a := range approvals() {
		if a.Deploy != "" {
			groups = append(groups, approvalGroup{group: a.key, message: a.Deploy, skipChildren: a.DeploySkipChildren})
		}
	}
	return groups
}

// destroyApprovalGroups returns the groups whose destroy message is declared on the type.
func destroyApprovalGroups() []approvalGroup {
	var groups []approvalGroup
	for _, a := range approvals() {
		if a.Destroy != "" {
			groups = append(groups, approvalGroup{group: a.key, message: a.Destroy})
		}
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

// approvalListing is a destroy message and the deletions it is printed above.
type approvalListing struct {
	message string
	actions []deployplan.Action
}

// destroyApprovalHook splits the deletions of one resource type by a destroy message that
// depends on the deployment state. It is registered for types whose [resources.Approval]
// sets DestroyFromState.
type destroyApprovalHook func(b *bundle.Bundle, deletes []deployplan.Action) ([]approvalListing, error)

var destroyApprovalHooks = map[string]destroyApprovalHook{}

func registerDestroyApprovalHook(key string, hook destroyApprovalHook) {
	if _, ok := destroyApprovalHooks[key]; ok {
		panic("destroy approval hook registered twice for " + key)
	}
	destroyApprovalHooks[key] = hook
}

// logDestroyApprovalsFromState prints the destroy messages that depend on the deployment
// state. The hooks run even if quiet is set, because reading the state can fail.
func logDestroyApprovalsFromState(ctx context.Context, b *bundle.Bundle, actions []deployplan.Action, quiet bool) error {
	for _, a := range approvals() {
		if !a.DestroyFromState {
			continue
		}
		hook, ok := destroyApprovalHooks[a.key]
		if !ok {
			return fmt.Errorf("internal error: no destroy approval hook is registered for %s", a.key)
		}
		listings, err := hook(b, filterGroup(actions, a.key, deployplan.Delete))
		if err != nil {
			return err
		}
		for _, l := range listings {
			if len(l.actions) == 0 || quiet {
				continue
			}
			cmdio.LogString(ctx, l.message)
			for _, action := range l.actions {
				cmdio.Log(ctx, action)
			}
			cmdio.LogString(ctx, "")
		}
	}
	return nil
}
