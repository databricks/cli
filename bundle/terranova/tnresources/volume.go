package tnresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceVolume struct {
	client *databricks.WorkspaceClient
	config catalog.CreateVolumeRequestContent
}

func NewResourceVolume(client *databricks.WorkspaceClient, schema *resources.Volume) (*ResourceVolume, error) {
	return &ResourceVolume{
		client: client,
		config: schema.CreateVolumeRequestContent,
	}, nil
}

func (r *ResourceVolume) Config() any {
	return r.config
}

func (r *ResourceVolume) DoRefresh(ctx context.Context, id string) (any, error) {
	return r.client.Volumes.ReadByName(ctx, id)
}

func (r *ResourceVolume) DoCreate(ctx context.Context) (string, any, error) {
	response, err := r.client.Volumes.Create(ctx, r.config)
	if err != nil {
		return "", nil, err
	}
	return response.FullName, response, nil
}

func (r *ResourceVolume) DoUpdate(ctx context.Context, id string) (any, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: r.config.Comment,
		Name:    id,
		NewName: "", // Not supported by Update(). Needs DoUpdateWithID()
		Owner:   "", // Not supported by DABs

		ForceSendFields: nil,
	}

	nameFromID, err := getNameFromID(id)
	if err != nil {
		return nil, err
	}

	if r.config.Name != nameFromID {
		return nil, fmt.Errorf("internal error: unexpected change of name from %#v to %#v", nameFromID, r.config.Name)
	}

	return r.client.Volumes.Update(ctx, updateRequest)
}

func (r *ResourceVolume) DoUpdateWithID(ctx context.Context, id string) (string, any, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: r.config.Comment,
		Name:    id,

		NewName: "", // Initialized below if needed
		Owner:   "", // Not supported by DABs

		ForceSendFields: nil,
	}

	items := strings.Split(id, ".")
	if len(items) == 0 {
		return "", nil, fmt.Errorf("unexpected id=%#v", id)
	}
	nameFromID := items[len(items)-1]

	if r.config.Name != nameFromID {
		updateRequest.NewName = r.config.Name
	}

	response, err := r.client.Volumes.Update(ctx, updateRequest)
	if err != nil || response == nil {
		return "", nil, err
	}

	return response.FullName, response, nil
}

func DeleteVolume(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	return client.Volumes.DeleteByName(ctx, id)
}

func (r *ResourceVolume) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	for _, change := range changes {
		if change.Path.String() == ".name" {
			return deployplan.ActionTypeUpdateWithID
		}
	}
	return deployplan.ActionTypeUpdate
}

func (r *ResourceVolume) WaitAfterCreate(ctx context.Context) (any, error) {
	// Intentional no-op
	return nil, nil
}

func (r *ResourceVolume) WaitAfterUpdate(ctx context.Context) (any, error) {
	// Intentional no-op
	return nil, nil
}

func getNameFromID(id string) (string, error) {
	items := strings.Split(id, ".")
	if len(items) == 0 {
		return "", fmt.Errorf("unexpected id=%#v", id)
	}
	return items[len(items)-1], nil
}
