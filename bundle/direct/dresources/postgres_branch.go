package dresources

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
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
	CreateTime *sdktime.Time          `json:"create_time,omitempty"`
	UpdateTime *sdktime.Time          `json:"update_time,omitempty"`
}

// Custom marshaler needed because embedded BranchSpec has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *PostgresBranchRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresBranchRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourcePostgresBranch struct {
	client *databricks.WorkspaceClient
}

type PostgresBranchState = resources.PostgresBranchConfig

func (*ResourcePostgresBranch) New(client *databricks.WorkspaceClient) *ResourcePostgresBranch {
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
		// it, so RemapState leaves it unset.
		ReplaceExisting: nil,

		// purge_on_delete is a delete-time query parameter; the GET API never
		// returns it, so RemapState leaves it unset.
		PurgeOnDelete: nil,

		BranchSpec: remote.BranchSpec,
	}
}

// makePostgresBranchRemote converts the SDK Branch into the embedded remote shape.
// Since 2026-11 GET echoes spec. Fields still marked spec:input_only in
// postgres_branches.generated.yml keep remote changes ignored until the OpenAPI
// annotations, and the config generated from them, catch up.
func makePostgresBranchRemote(branch *postgres.Branch) *PostgresBranchRemote {
	var spec postgres.BranchSpec
	if branch.Spec != nil {
		spec = *branch.Spec
	}
	return &PostgresBranchRemote{
		BranchSpec: spec,
		BranchId:   branch.BranchId,
		Parent:     branch.Parent,
		Name:       branch.Name,
		Status:     branch.Status,
		Uid:        branch.Uid,
		CreateTime: branch.CreateTime,
		UpdateTime: branch.UpdateTime,
	}
}

func (r *ResourcePostgresBranch) DoRead(ctx context.Context, id string) (*PostgresBranchRemote, error) {
	branch, err := r.client.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresBranchRemote(branch), nil
}

func (r *ResourcePostgresBranch) DoCreate(ctx context.Context, config *PostgresBranchState) (string, *PostgresBranchRemote, error) {
	req := postgres.CreateBranchRequest{
		BranchId: config.BranchId,
		Parent:   config.Parent,
		Branch: postgres.Branch{
			Spec: &config.BranchSpec,

			// Output-only fields.
			BranchId:        "",
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		ReplaceExisting: config.ReplaceExisting != nil && *config.ReplaceExisting,
		ForceSendFields: nil,
	}
	start := time.Now()
	waiter, err := r.client.Postgres.CreateBranch(ctx, req)
	if err != nil {
		return r.recoverFailedCreate(ctx, req, start, err)
	}
	remote, err := r.waitCreated(ctx, waiter)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

// waitCreated waits for the create operation of a branch that a create request returned.
func (r *ResourcePostgresBranch) waitCreated(ctx context.Context, waiter postgres.CreateBranchOperationInterface) (*PostgresBranchRemote, error) {
	result, err := waitCreateOperation(ctx, waiter.Wait)
	if err != nil {
		return nil, err
	}
	return makePostgresBranchRemote(result), nil
}

// Tuning for waitBranchReady, variables so that tests can shorten them.
var (
	// branchReadyTimeout bounds the wait for a recovered branch that is still in INIT.
	branchReadyTimeout = 10 * time.Minute
	// branchReadyPollInterval is the pause between reads of a recovered branch in INIT.
	branchReadyPollInterval = 5 * time.Second
)

// recoverFailedCreate handles a create request that failed; see recoverCreate.
func (r *ResourcePostgresBranch) recoverFailedCreate(ctx context.Context, req postgres.CreateBranchRequest, start time.Time, createErr error) (string, *PostgresBranchRemote, error) {
	// A branch found after a replace may predate it, and repeating a replace is safe.
	if req.ReplaceExisting {
		return "", nil, retrySafe(createErr)
	}

	name := req.Parent + "/branches/" + req.BranchId
	remote, err := recoverCreate(ctx, createRecovery[postgres.Branch, *PostgresBranchRemote]{
		name:  "branch " + name,
		start: start,
		get: func(ctx context.Context) (*postgres.Branch, error) {
			return r.client.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: name})
		},
		createTime:      func(found *postgres.Branch) (time.Time, bool) { return sdkTime(found.CreateTime) },
		alreadyExists:   isPostgresAlreadyExists,
		createdByCaller: nil,
		use: func(ctx context.Context, branch *postgres.Branch) (*PostgresBranchRemote, error) {
			return r.useRecoveredBranch(ctx, branch, createErr)
		},
		create: func(ctx context.Context) (func() (*PostgresBranchRemote, error), error) {
			waiter, err := r.client.Postgres.CreateBranch(ctx, req)
			if err != nil {
				return nil, err
			}
			return func() (*PostgresBranchRemote, error) { return r.waitCreated(ctx, waiter) }, nil
		},
	}, createErr)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

func readBranch(ctx context.Context, client *databricks.WorkspaceClient, name string) (*postgres.Branch, error) {
	return retry.With(ctx, retry.IsTransientRead, func() (*postgres.Branch, error) {
		return client.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: name})
	})
}

// useRecoveredBranch returns the branch recoverFailedCreate found, once it is READY.
//
// The wait is inline rather than in WaitAfterCreate, like the wait for the create operation on the
// success path: a WaitAfterCreate would read every created branch once more. Once the branch is
// known to be ours, it is returned even when the wait ends early, so that state records it and the
// next deploy does not fail with ALREADY_EXISTS.
func (r *ResourcePostgresBranch) useRecoveredBranch(ctx context.Context, branch *postgres.Branch, createErr error) (*PostgresBranchRemote, error) {
	branch, err := waitBranchReady(ctx, r.client, branch)
	if err != nil {
		return nil, err
	}

	// Other states, such as DELETED or ARCHIVED, will not become READY.
	if state := branchState(branch); state != postgres.BranchStatusStateReady && state != postgres.BranchStatusStateInit {
		return nil, fmt.Errorf("%w; branch %s exists in state %q", createErr, branch.Name, state)
	}
	return makePostgresBranchRemote(branch), nil
}

// waitBranchReady polls branch while it is in INIT, up to branchReadyTimeout, and returns the last
// branch it read. A timeout or a failed read ends the wait early with a warning; only a done
// context is returned as an error.
func waitBranchReady(ctx context.Context, client *databricks.WorkspaceClient, branch *postgres.Branch) (*postgres.Branch, error) {
	deadline := time.Now().Add(branchReadyTimeout)
	for branchState(branch) == postgres.BranchStatusStateInit {
		if !time.Now().Before(deadline) {
			log.Warnf(ctx, "branch %s is still initializing after %s", branch.Name, branchReadyTimeout)
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(branchReadyPollInterval):
		}
		next, err := readBranch(ctx, client, branch.Name)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			log.Warnf(ctx, "waiting for branch %s to become ready: %s", branch.Name, err)
			break
		}
		branch = next
	}
	return branch, nil
}

func branchState(branch *postgres.Branch) postgres.BranchStatusState {
	if branch.Status == nil {
		return ""
	}
	return branch.Status.CurrentState
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

	waiter, err := r.client.Postgres.UpdateBranch(ctx, postgres.UpdateBranchRequest{
		Branch: postgres.Branch{
			Spec: &config.BranchSpec,

			// Output-only fields.
			BranchId:        "",
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		Name: id,
		UpdateMask: fieldmask.FieldMask{
			Paths: fieldPaths,
		},
	})
	if err != nil {
		return nil, err
	}

	// Wait for the update to complete
	result, err := waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return makePostgresBranchRemote(result), nil
}

func (r *ResourcePostgresBranch) DoDelete(ctx context.Context, id string, state *PostgresBranchState) error {
	waiter, err := r.client.Postgres.DeleteBranch(ctx, postgres.DeleteBranchRequest{
		Name:            id,
		Purge:           state.PurgeOnDelete != nil && *state.PurgeOnDelete,
		ForceSendFields: nil,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
