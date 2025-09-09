package tnresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceVolume struct {
	client *databricks.WorkspaceClient
}

func (*ResourceVolume) New(client *databricks.WorkspaceClient) *ResourceVolume {
	return &ResourceVolume{client: client}
}

func (*ResourceVolume) PrepareState(input *resources.Volume) *catalog.CreateVolumeRequestContent {
	return &input.CreateVolumeRequestContent
}

func (r *ResourceVolume) DoRefresh(ctx context.Context, id string) (*catalog.VolumeInfo, error) {
	return r.client.Volumes.ReadByName(ctx, id)
}

func (r *ResourceVolume) DoCreate(ctx context.Context, config *catalog.CreateVolumeRequestContent) (string, *catalog.VolumeInfo, error) {
	response, err := r.client.Volumes.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return response.FullName, response, nil
}

func (r *ResourceVolume) DoUpdate(ctx context.Context, id string, config *catalog.CreateVolumeRequestContent) (*catalog.VolumeInfo, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: config.Comment,
		Name:    id,
		NewName: "", // Not supported by Update(). Needs DoUpdateWithID()
		Owner:   "", // Not supported by DABs

		ForceSendFields: nil,
	}

	nameFromID, err := getNameFromID(id)
	if err != nil {
		return nil, err
	}

	if config.Name != nameFromID {
		return nil, fmt.Errorf("internal error: unexpected change of name from %#v to %#v", nameFromID, config.Name)
	}

	response, err := r.client.Volumes.Update(ctx, updateRequest)
	if err != nil {
		return nil, err
	}

	if id != response.FullName {
		log.Warnf(ctx, "volumes: response contains unexpected full_name=%#v (expected %#v)", response.FullName, id)
	}

	return response, err
}

func (r *ResourceVolume) DoUpdateWithID(ctx context.Context, id string, config *catalog.CreateVolumeRequestContent) (string, *catalog.VolumeInfo, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: config.Comment,
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

	if config.Name != nameFromID {
		updateRequest.NewName = config.Name
	}

	response, err := r.client.Volumes.Update(ctx, updateRequest)
	if err != nil || response == nil {
		return "", nil, err
	}

	return response.FullName, response, nil
}

func (r *ResourceVolume) DoDelete(ctx context.Context, id string) error {
	return r.client.Volumes.DeleteByName(ctx, id)
}

func (*ResourceVolume) RecreateFields() []string {
	return []string{
		".catalog_name",
		".schema_name",
		".storage_location",
		".volume_type",
	}
}

func (r *ResourceVolume) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	for _, change := range changes {
		if change.Path.String() == ".name" {
			return deployplan.ActionTypeUpdateWithID
		}
	}
	return deployplan.ActionTypeUpdate
}

func getNameFromID(id string) (string, error) {
	items := strings.Split(id, ".")
	if len(items) == 0 {
		return "", fmt.Errorf("unexpected id=%#v", id)
	}
	return items[len(items)-1], nil
}
