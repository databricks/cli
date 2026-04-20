package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/retries"
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
	// Kick off the create request. Wait for the space to become active in
	// WaitAfterCreate so that parallel creates are not blocked here.
	_, err := r.client.Apps.CreateSpace(ctx, apps.CreateSpaceRequest{
		Space: *config,
	})
	if err != nil {
		return "", nil, err
	}
	return config.Name, nil, nil
}

func (r *ResourceAppSpace) DoUpdate(ctx context.Context, id string, config *apps.Space, _ *PlanEntry) (*apps.Space, error) {
	_, err := r.client.Apps.UpdateSpace(ctx, apps.UpdateSpaceRequest{
		Name:       id,
		Space:      *config,
		UpdateMask: fieldmask.FieldMask{Paths: []string{"description", "resources", "user_api_scopes", "usage_policy_id"}},
	})
	return nil, err
}

func (r *ResourceAppSpace) WaitAfterCreate(ctx context.Context, config *apps.Space) (*apps.Space, error) {
	return r.waitForSpaceActive(ctx, config.Name)
}

func (r *ResourceAppSpace) WaitAfterUpdate(ctx context.Context, config *apps.Space) (*apps.Space, error) {
	return r.waitForSpaceActive(ctx, config.Name)
}

func (r *ResourceAppSpace) waitForSpaceActive(ctx context.Context, name string) (*apps.Space, error) {
	retrier := retries.New[apps.Space](retries.WithTimeout(20 * time.Minute))
	return retrier.Run(ctx, func(ctx context.Context) (*apps.Space, error) {
		space, err := r.client.Apps.GetSpace(ctx, apps.GetSpaceRequest{Name: name})
		if err != nil {
			return nil, retries.Halt(err)
		}
		if space.Status == nil {
			return nil, retries.Continues("waiting for status")
		}
		switch space.Status.State {
		case apps.SpaceStatusSpaceStateSpaceActive:
			return space, nil
		case apps.SpaceStatusSpaceStateSpaceError:
			return nil, retries.Halt(fmt.Errorf("space %s is in ERROR state: %s", name, space.Status.Message))
		default:
			return nil, retries.Continues(fmt.Sprintf("space state: %s", space.Status.State))
		}
	})
}

func (r *ResourceAppSpace) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Apps.DeleteSpace(ctx, apps.DeleteSpaceRequest{Name: id})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
