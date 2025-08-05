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

func (r *ResourceVolume) DoCreate(ctx context.Context) (string, error) {
	response, err := r.client.Volumes.Create(ctx, r.config)
	if err != nil {
		return "", SDKError{Method: "Volumes.Create", Err: err}
	}
	return response.FullName, nil
}

func (r *ResourceVolume) DoUpdate(ctx context.Context, id string) error {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: r.config.Comment,
		Name:    id,
	}

	nameFromID, err := getNameFromID(id)
	if err != nil {
		return err
	}

	if r.config.Name != nameFromID {
		return fmt.Errorf("internal error: unexpected change of name from %#v to %#v", nameFromID, r.config.Name)
	}

	response, err := r.client.Volumes.Update(ctx, updateRequest)
	if err != nil {
		return SDKError{Method: "Volumes.Update", Err: err}
	}

	if id != response.FullName {
		log.Warnf(ctx, "volumes: response contains unexpected full_name=%#v (expected %#v)", response.FullName, id)
	}

	return nil
}

func (r *ResourceVolume) DoUpdateWithID(ctx context.Context, id string) (string, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: r.config.Comment,
		Name:    id,
	}

	items := strings.Split(id, ".")
	if len(items) == 0 {
		return "", fmt.Errorf("unexpected id=%#v", id)
	}
	nameFromID := items[len(items)-1]

	if r.config.Name != nameFromID {
		updateRequest.NewName = r.config.Name
	}

	response, err := r.client.Volumes.Update(ctx, updateRequest)
	if err != nil {
		return "", SDKError{Method: "Volumes.Update", Err: err}
	}

	return response.FullName, nil
}

func DeleteVolume(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	err := client.Volumes.DeleteByName(ctx, id)
	if err != nil {
		return SDKError{Method: "Volumes.Delete", Err: err}
	}
	return nil
}

func (r *ResourceVolume) WaitAfterCreate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func (r *ResourceVolume) WaitAfterUpdate(ctx context.Context) error {
	// Intentional no-op
	return nil
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
