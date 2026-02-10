package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresBranch struct {
	client *databricks.WorkspaceClient
}

type PostgresBranchState = resources.PostgresBranchConfig

func (*ResourcePostgresBranch) New(client *databricks.WorkspaceClient) *ResourcePostgresBranch {
	return &ResourcePostgresBranch{client: client}
}

func (*ResourcePostgresBranch) PrepareState(input *resources.PostgresBranch) *PostgresBranchState {
	return &PostgresBranchState{
		BranchId:   input.BranchId,
		Parent:     input.Parent,
		BranchSpec: input.BranchSpec,
	}
}

func (*ResourcePostgresBranch) RemapState(remote *postgres.Branch) *PostgresBranchState {
	// Extract branch_id from hierarchical name: "projects/{project_id}/branches/{branch_id}"
	// TODO: log error when we have access to the context
	components, _ := ParsePostgresName(remote.Name)

	return &PostgresBranchState{
		BranchId: components.BranchID,
		Parent:   remote.Parent,

		// The read API does not return the spec, only the status.
		// This means we cannot detect remote drift for spec fields.
		// Use an empty struct (not nil) so field-level diffing works correctly.
		BranchSpec: postgres.BranchSpec{
			ExpireTime:       nil,
			IsProtected:      false,
			NoExpiry:         false,
			SourceBranch:     "",
			SourceBranchLsn:  "",
			SourceBranchTime: nil,
			Ttl:              nil,
			ForceSendFields:  nil,
		},
	}
}

func (r *ResourcePostgresBranch) DoRead(ctx context.Context, id string) (*postgres.Branch, error) {
	return r.client.Postgres.GetBranch(ctx, postgres.GetBranchRequest{Name: id})
}

func (r *ResourcePostgresBranch) DoCreate(ctx context.Context, config *PostgresBranchState) (string, *postgres.Branch, error) {
	waiter, err := r.client.Postgres.CreateBranch(ctx, postgres.CreateBranchRequest{
		BranchId: config.BranchId,
		Parent:   config.Parent,
		Branch: postgres.Branch{
			Spec: &config.BranchSpec,

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
	// The Postgres API requires that fields in the update_mask must be present in the request body.
	// When a field is removed from config (nil in Go), it doesn't appear in the JSON payload,
	// but would appear in a granular update_mask, causing the backend to reject the request.
	// To avoid this, we use "spec" (which is always present) instead of listing individual fields.
	waiter, err := r.client.Postgres.UpdateBranch(ctx, postgres.UpdateBranchRequest{
		Branch: postgres.Branch{
			Spec: &config.BranchSpec,

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
			Paths: []string{"spec"},
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
