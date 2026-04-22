package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type ResourceAppSpace struct {
	client *databricks.WorkspaceClient
}

func (*ResourceAppSpace) New(client *databricks.WorkspaceClient) *ResourceAppSpace {
	return &ResourceAppSpace{client: client}
}

func (*ResourceAppSpace) PrepareState(input *resources.AppSpace) *apps.Space {
	return &input.Space
}

func (r *ResourceAppSpace) DoRead(ctx context.Context, id string) (*apps.Space, error) {
	return r.client.Apps.GetSpace(ctx, apps.GetSpaceRequest{Name: id})
}

func (r *ResourceAppSpace) DoCreate(ctx context.Context, config *apps.Space) (string, *apps.Space, error) {
	waiter, err := r.client.Apps.CreateSpace(ctx, apps.CreateSpaceRequest{
		Space: *config,
	})
	if err != nil {
		return "", nil, err
	}
	space, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}
	return space.Name, space, nil
}

func (r *ResourceAppSpace) DoUpdate(ctx context.Context, id string, config *apps.Space, _ *PlanEntry) (*apps.Space, error) {
	waiter, err := r.client.Apps.UpdateSpace(ctx, apps.UpdateSpaceRequest{
		Name:       id,
		Space:      *config,
		UpdateMask: fieldmask.FieldMask{Paths: []string{"description", "resources", "user_api_scopes", "usage_policy_id"}},
	})
	if err != nil {
		return nil, err
	}
	return waiter.Wait(ctx)
}

func (r *ResourceAppSpace) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Apps.DeleteSpace(ctx, apps.DeleteSpaceRequest{Name: id})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
