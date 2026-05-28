package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
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

		BranchSpec: remote.BranchSpec,
	}
}

// makePostgresBranchRemote converts the SDK Branch into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and resources.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresBranchRemote(branch *postgres.Branch) *PostgresBranchRemote {
	var spec postgres.BranchSpec
	if branch.Spec != nil {
		spec = *branch.Spec
	}
	var branchID string
	if branch.Status != nil {
		branchID = branch.Status.BranchId
	}
	return &PostgresBranchRemote{
		BranchSpec: spec,
		BranchId:   branchID,
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

func (r *ResourcePostgresBranch) DoCreate(ctx context.Context, _ *Engine, config *PostgresBranchState) (string, *PostgresBranchRemote, error) {
	waiter, err := r.client.Postgres.CreateBranch(ctx, postgres.CreateBranchRequest{
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
		ReplaceExisting: config.ReplaceExisting,
		ForceSendFields: nil,
	})
	if err != nil {
		return "", nil, err
	}

	// Wait for the branch to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	remote := makePostgresBranchRemote(result)
	return remote.Name, remote, nil
}

func (r *ResourcePostgresBranch) DoUpdate(ctx context.Context, _ *Engine, id string, config *PostgresBranchState, entry *PlanEntry) (*PostgresBranchRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Branch object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.")

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

func (r *ResourcePostgresBranch) DoDelete(ctx context.Context, id string, _ *PostgresBranchState) error {
	waiter, err := r.client.Postgres.DeleteBranch(ctx, postgres.DeleteBranchRequest{
		Name:            id,
		Purge:           false,
		ForceSendFields: nil,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
