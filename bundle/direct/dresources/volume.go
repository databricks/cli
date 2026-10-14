package dresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	volumes "github.com/databricks/cli/libs/sdktypes/uc/volumes/v1"
	"github.com/databricks/cli/libs/utils"
	sdkvolumes "github.com/databricks/sdk-go/uc/volumes/v1"
)

type ResourceVolume struct {
	client *sdkclients.Clients
}

func (*ResourceVolume) New(client *sdkclients.Clients) *ResourceVolume {
	return &ResourceVolume{client: client}
}

func (*ResourceVolume) PrepareState(input *resources.Volume) *resources.CreateVolumeRequestContent {
	return &input.CreateVolumeRequestContent
}

func (*ResourceVolume) RemapState(info *volumes.VolumeInfo) *resources.CreateVolumeRequestContent {
	return &resources.CreateVolumeRequestContent{
		CatalogName:     info.CatalogName,
		Comment:         info.Comment,
		Name:            info.Name,
		SchemaName:      info.SchemaName,
		StorageLocation: info.StorageLocation,
		VolumeType:      info.VolumeType,
	}
}

func (r *ResourceVolume) DoRead(ctx context.Context, id string) (*volumes.VolumeInfo, error) {
	info, err := r.client.Volumes.GetVolume(ctx, sdkvolumes.GetVolumeRequest{FullNameArg: new(id)})
	if err != nil {
		return nil, err
	}
	return volumes.VolumeInfoFromSDK(info)
}

func (r *ResourceVolume) DoCreate(ctx context.Context, config *resources.CreateVolumeRequestContent) (string, *volumes.VolumeInfo, error) {
	request, err := toSDK[sdkvolumes.CreateVolumeRequest](config.AsRequest())
	if err != nil {
		return "", nil, err
	}
	response, err := r.client.Volumes.CreateVolume(ctx, request)
	if err != nil {
		return "", nil, err
	}
	info, err := volumes.VolumeInfoFromSDK(response)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(info.FullName), info, nil
}

// comment is always sent, see ResourceSchema.DoUpdate. Verified against a real workspace: {"comment": ""} clears it.
// owner is not supported by DABs.
func (r *ResourceVolume) update(ctx context.Context, id string, config *resources.CreateVolumeRequestContent, newName *string) (*volumes.VolumeInfo, error) {
	request, err := toSDK[sdkvolumes.UpdateVolumeRequest](&volumes.UpdateVolumeRequest{
		FullNameArg: new(id),
		NewName:     newName,
		Comment:     orZero(config.Comment),
	})
	if err != nil {
		return nil, err
	}
	response, err := r.client.Volumes.UpdateVolume(ctx, request)
	if err != nil {
		return nil, err
	}
	return volumes.VolumeInfoFromSDK(response)
}

func (r *ResourceVolume) DoUpdate(ctx context.Context, id string, config *resources.CreateVolumeRequestContent, _ *PlanEntry) (*volumes.VolumeInfo, error) {
	nameFromID, err := getNameFromID(id)
	if err != nil {
		return nil, err
	}

	if name := utils.Deref(config.Name); name != nameFromID {
		return nil, fmt.Errorf("internal error: unexpected change of name from %#v to %#v", nameFromID, name)
	}

	// new_name is not supported by Update(), it needs DoUpdateWithID().
	response, err := r.update(ctx, id, config, nil)
	if err != nil {
		return nil, err
	}

	if fullName := utils.Deref(response.FullName); id != fullName {
		log.Warnf(ctx, "volumes: response contains unexpected full_name=%#v (expected %#v)", fullName, id)
	}

	return response, nil
}

func (r *ResourceVolume) DoUpdateWithID(ctx context.Context, id string, config *resources.CreateVolumeRequestContent, _ *PlanEntry) (string, *volumes.VolumeInfo, error) {
	nameFromID, err := getNameFromID(id)
	if err != nil {
		return "", nil, err
	}

	var newName *string
	if name := utils.Deref(config.Name); name != nameFromID {
		newName = new(name)
	}

	response, err := r.update(ctx, id, config, newName)
	if err != nil || response == nil {
		return "", nil, err
	}

	return utils.Deref(response.FullName), response, nil
}

func (r *ResourceVolume) DoDelete(ctx context.Context, id string, _ *resources.CreateVolumeRequestContent) error {
	_, err := r.client.Volumes.DeleteVolume(ctx, sdkvolumes.DeleteVolumeRequest{FullNameArg: new(id)})
	return err
}

func getNameFromID(id string) (string, error) {
	items := strings.Split(id, ".")
	if len(items) == 0 {
		return "", fmt.Errorf("unexpected id=%#v", id)
	}
	return items[len(items)-1], nil
}
