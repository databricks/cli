package dresources

import (
	"context"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
)

// PostgresBranchRemote is the return type for DoRead. It embeds BranchSpec so that
// all paths in StateType are valid paths in RemoteType, enabling drift detection
// for spec fields once the backend echoes spec on GET.
type PostgresBranchRemote struct {
	postgres.BranchSpec

	BranchId string `json:"branch_id,omitempty"`
	Parent   string `json:"parent,omitempty"`

	Name       string                 `json:"name,omitempty"`
	Status     *postgres.BranchStatus `json:"status,omitempty"`
	Uid        string                 `json:"uid,omitempty"`
	CreateTime *types.Time            `json:"create_time,omitempty"`
	UpdateTime *types.Time            `json:"update_time,omitempty"`
}

type ResourcePostgresBranch struct {
	client *sdkclients.Clients
}

type PostgresBranchState = resources.PostgresBranchConfig

func (*ResourcePostgresBranch) New(client *sdkclients.Clients) *ResourcePostgresBranch {
	return &ResourcePostgresBranch{client: client}
}

func (*ResourcePostgresBranch) PrepareState(input *resources.PostgresBranch) *PostgresBranchState {
	return &PostgresBranchState{
		BranchId:        input.BranchId,
		Parent:          input.Parent,
		ReplaceExisting: input.ReplaceExisting,
		PurgeOnDelete:   input.PurgeOnDelete,
		BranchSpec:      input.BranchSpec,
	}
}

func (*ResourcePostgresBranch) RemapState(remote *PostgresBranchRemote) *PostgresBranchState {
	return &PostgresBranchState{
		BranchId: remote.BranchId,
		Parent:   remote.Parent,

		// replace_existing is a create-time-only flag; the GET API never returns
		// it, so RemapState leaves it false.
		ReplaceExisting: false,

		// purge_on_delete is a delete-time query parameter; the GET API never
		// returns it, so RemapState leaves it unset.
		PurgeOnDelete: nil,

		BranchSpec: remote.BranchSpec,
	}
}

// makePostgresBranchRemote converts the SDK Branch into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and postgres_branches.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresBranchRemote(resp *sdkpostgres.Branch) (*PostgresBranchRemote, error) {
	branch, err := postgres.BranchFromSDK(resp)
	if err != nil {
		return nil, err
	}
	var spec postgres.BranchSpec
	if branch.Spec != nil {
		spec = *branch.Spec
	}
	return &PostgresBranchRemote{
		BranchSpec: spec,
		BranchId:   utils.Deref(branch.BranchId),
		Parent:     utils.Deref(branch.Parent),
		Name:       utils.Deref(branch.Name),
		Status:     branch.Status,
		Uid:        utils.Deref(branch.Uid),
		CreateTime: branch.CreateTime,
		UpdateTime: branch.UpdateTime,
	}, nil
}

func (r *ResourcePostgresBranch) DoRead(ctx context.Context, id string) (*PostgresBranchRemote, error) {
	branch, err := r.client.Postgres.GetBranch(ctx, sdkpostgres.GetBranchRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresBranchRemote(branch)
}

func (r *ResourcePostgresBranch) DoCreate(ctx context.Context, config *PostgresBranchState) (string, *PostgresBranchRemote, error) {
	// Only spec is set; the other Branch fields are output-only.
	request, err := postgres.CreateBranchRequestToSDK(&postgres.CreateBranchRequest{
		BranchId: nonEmpty(config.BranchId),
		Parent:   nonEmpty(config.Parent),
		Branch: &postgres.Branch{
			Spec:       &config.BranchSpec,
			BranchId:   nil, // Not set by DABs
			CreateTime: nil, // Not set by DABs
			Name:       nil, // Not set by DABs
			Parent:     nil, // Not set by DABs
			Status:     nil, // Not set by DABs
			Uid:        nil, // Not set by DABs
			UpdateTime: nil, // Not set by DABs
		},
		ReplaceExisting: trueOrNil(config.ReplaceExisting),
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Postgres.CreateBranch(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	// Wait for the branch to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	remote, err := makePostgresBranchRemote(result)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

// expire_time, no_expiry and ttl are three sides of one oneof, and the API accepts them
// in update_mask only under the group name: masking the field itself is answered with
// "Unknown field path in update_mask". Probed against a real workspace on 2026-08-31.
var branchOneofGroups = map[string]string{
	"expire_time": "expiration",
	"no_expiry":   "expiration",
	"ttl":         "expiration",
}

func (r *ResourcePostgresBranch) DoUpdate(ctx context.Context, id string, config *PostgresBranchState, entry *PlanEntry) (*PostgresBranchRemote, error) {
	// Build the mask from the plan's change list and prefix with "spec." (the
	// API expects paths relative to Branch). The API rejects mask entries
	// that aren't also populated in the request body, and a wildcard "*"
	// expands to nested attributes the body would have to set too — so we
	// can't use a static all-fields mask. The change list naturally tracks
	// what the user actually set, so the body and mask stay consistent.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.", branchOneofGroups)

	// purge_on_delete is an input-only flag consulted at delete time; it is
	// not a spec field. Strip it from the mask so toggling it between deploys
	// becomes a state-only refresh (the framework saves newState when this
	// returns nil error).
	fieldPaths = slices.DeleteFunc(fieldPaths, func(p string) bool {
		return p == "spec.purge_on_delete"
	})
	if len(fieldPaths) == 0 {
		return nil, nil
	}

	// Only spec is set; the other Branch fields are output-only.
	updated, err := updatePostgres[postgres.Branch](ctx, r.client, id, strings.Join(fieldPaths, ","), &postgres.Branch{
		Spec:       &config.BranchSpec,
		BranchId:   nil, // Not set by DABs
		CreateTime: nil, // Not set by DABs
		Name:       nil, // Not set by DABs
		Parent:     nil, // Not set by DABs
		Status:     nil, // Not set by DABs
		Uid:        nil, // Not set by DABs
		UpdateTime: nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}
	result, err := postgres.BranchToSDK(updated)
	if err != nil {
		return nil, err
	}
	return makePostgresBranchRemote(result)
}

func (r *ResourcePostgresBranch) DoDelete(ctx context.Context, id string, state *PostgresBranchState) error {
	waiter, err := r.client.Postgres.DeleteBranch(ctx, sdkpostgres.DeleteBranchRequest{
		Name:  new(id),
		Purge: trueOrNil(utils.Deref(state.PurgeOnDelete)),
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
