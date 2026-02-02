package dresources

import (
	"context"
	"errors"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresBranch struct {
	client *databricks.WorkspaceClient
}

// PostgresBranchState contains only the fields needed for creation/update.
// It does NOT include output-only fields like Name, which are only available after API response.
type PostgresBranchState struct {
	BranchId string               `json:"branch_id,omitempty"`
	Parent   string               `json:"parent,omitempty"`
	Spec     *postgres.BranchSpec `json:"spec,omitempty"`
}

func (*ResourcePostgresBranch) New(client *databricks.WorkspaceClient) *ResourcePostgresBranch {
	return &ResourcePostgresBranch{client: client}
}

func (*ResourcePostgresBranch) PrepareState(input *resources.PostgresBranch) *PostgresBranchState {
	return &PostgresBranchState{
		BranchId: input.BranchId,
		Parent:   input.Parent,
		Spec:     &input.BranchSpec,
	}
}

func (*ResourcePostgresBranch) RemapState(remote *postgres.Branch) *PostgresBranchState {
	// Extract branch_id from hierarchical name: "projects/{project_id}/branches/{branch_id}"
	branchId := ""
	if remote.Name != "" {
		parts := strings.Split(remote.Name, "/")
		if len(parts) >= 4 {
			branchId = parts[3]
		}
	}

	return &PostgresBranchState{
		BranchId: branchId,
		Parent:   remote.Parent,

		// The read API does not return the spec, only the status.
		// This means we cannot detect remote drift for spec fields.
		Spec: nil,
	}
}

func (r *ResourcePostgresBranch) DoRead(ctx context.Context, id string) (*postgres.Branch, error) {
	return r.client.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: id})
}

func (r *ResourcePostgresBranch) DoCreate(ctx context.Context, config *PostgresBranchState) (string, *postgres.Branch, error) {
	branchId := config.BranchId
	if branchId == "" {
		return "", nil, errors.New("branch_id must be specified")
	}

	parent := config.Parent
	if parent == "" {
		return "", nil, errors.New("parent (project name) must be specified")
	}

	waiter, err := r.client.Postgres.CreateBranch(ctx, postgres.CreateBranchRequest{
		BranchId: branchId,
		Parent:   parent,
		Branch: postgres.Branch{
			Spec: config.Spec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
	})
	if err != nil {
		return "", nil, err
	}

	// Wait for the branch to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	return result.Name, result, nil
}

func (r *ResourcePostgresBranch) DoUpdate(ctx context.Context, id string, config *PostgresBranchState, changes Changes) (*postgres.Branch, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	fieldPaths := collectUpdatePaths(changes)

	waiter, err := r.client.Postgres.UpdateBranch(ctx, postgres.UpdateBranchRequest{
		Branch: postgres.Branch{
			Spec: config.Spec,

			// Output-only fields.
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
	return result, err
}

func (r *ResourcePostgresBranch) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Postgres.DeleteBranch(ctx, postgres.DeleteBranchRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
