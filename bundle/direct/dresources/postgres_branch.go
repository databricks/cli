package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresBranch struct {
	client *databricks.WorkspaceClient
}

// PostgresBranchState wraps the postgres.Branch with additional fields needed for creation.
type PostgresBranchState struct {
	postgres.Branch
	BranchId string `json:"branch_id,omitempty"`
}

func (*ResourcePostgresBranch) New(client *databricks.WorkspaceClient) *ResourcePostgresBranch {
	return &ResourcePostgresBranch{client: client}
}

func (*ResourcePostgresBranch) PrepareState(input *resources.PostgresBranch) *PostgresBranchState {
	return &PostgresBranchState{
		Branch:   input.Branch,
		BranchId: input.BranchId,
	}
}

func (*ResourcePostgresBranch) RemapState(remote *postgres.Branch) *PostgresBranchState {
	return &PostgresBranchState{
		Branch: *remote,
		// BranchId is not available in remote state, it's already part of the Name
	}
}

func (r *ResourcePostgresBranch) DoRead(ctx context.Context, id string) (*postgres.Branch, error) {
	return r.client.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: id})
}

func (r *ResourcePostgresBranch) DoCreate(ctx context.Context, config *PostgresBranchState) (string, *postgres.Branch, error) {
	branchId := config.BranchId
	if branchId == "" {
		return "", nil, fmt.Errorf("branch_id must be specified")
	}

	parent := config.Parent
	if parent == "" {
		return "", nil, fmt.Errorf("parent (project name) must be specified")
	}

	waiter, err := r.client.Postgres.CreateBranch(ctx, postgres.CreateBranchRequest{
		BranchId: branchId,
		Parent:   parent,
		Branch:   config.Branch,
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

func (r *ResourcePostgresBranch) DoUpdate(ctx context.Context, id string, config *PostgresBranchState, _ Changes) (*postgres.Branch, error) {
	waiter, err := r.client.Postgres.UpdateBranch(ctx, postgres.UpdateBranchRequest{
		Branch: config.Branch,
		Name:   id,
		UpdateMask: fieldmask.FieldMask{
			Paths: []string{"*"},
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
