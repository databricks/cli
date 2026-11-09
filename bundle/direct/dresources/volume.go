package dresources

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// VolumeState is the volume's config plus its volume_id. The backend assigns volume_id and a
// rename keeps it, so recording it lets the plan recognize the volume under a new name (see
// InternalID).
type VolumeState struct {
	catalog.CreateVolumeRequestContent
	VolumeId string `json:"volume_id,omitempty"`
}

func (s *VolumeState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s VolumeState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceVolume struct {
	client *databricks.WorkspaceClient
}

func (*ResourceVolume) New(client *databricks.WorkspaceClient) *ResourceVolume {
	return &ResourceVolume{client: client}
}

func (*ResourceVolume) PrepareState(input *resources.Volume) *VolumeState {
	return &VolumeState{
		CreateVolumeRequestContent: input.CreateVolumeRequestContent,
		VolumeId:                   "", // Recorded from the backend when the volume is created or updated.
	}
}

func (*ResourceVolume) RemapState(info *catalog.VolumeInfo) *VolumeState {
	return &VolumeState{
		CatalogName:     info.CatalogName,
		Comment:         info.Comment,
		Name:            info.Name,
		SchemaName:      info.SchemaName,
		StorageLocation: info.StorageLocation,
		VolumeType:      info.VolumeType,
		ForceSendFields: utils.FilterFields[catalog.CreateVolumeRequestContent](info.ForceSendFields),
		VolumeId:        info.VolumeId,
	}
}

func (r *ResourceVolume) DoRead(ctx context.Context, id string) (*catalog.VolumeInfo, error) {
	return r.client.Volumes.ReadByName(ctx, id)
}

func (r *ResourceVolume) DoCreate(ctx context.Context, config *VolumeState) (string, *catalog.VolumeInfo, error) {
	response, err := r.client.Volumes.Create(ctx, config.CreateVolumeRequestContent)
	if err != nil {
		return "", nil, err
	}
	config.VolumeId = response.VolumeId
	return response.FullName, response, nil
}

// See schemaForceSend. Verified against a real workspace: {"comment": ""} clears it.
var volumeForceSend = []string{"Comment"}

func (r *ResourceVolume) DoUpdate(ctx context.Context, id string, config *VolumeState, _ *PlanEntry) (*catalog.VolumeInfo, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: config.Comment,
		Name:    id,
		NewName: "", // Not supported by Update(). Needs DoUpdateWithID()
		Owner:   "", // Not supported by DABs

		ForceSendFields: utils.FilterFields[catalog.UpdateVolumeRequestContent](append(slices.Clone(volumeForceSend), config.ForceSendFields...), "NewName", "Owner"),
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

	config.VolumeId = response.VolumeId
	return response, err
}

func (r *ResourceVolume) DoUpdateWithID(ctx context.Context, id string, config *VolumeState, entry *PlanEntry) (string, *catalog.VolumeInfo, error) {
	updateRequest := catalog.UpdateVolumeRequestContent{
		Comment: config.Comment,
		Name:    "", // Initialized below

		NewName: "", // Initialized below if needed
		Owner:   "", // Not supported by DABs

		ForceSendFields: utils.FilterFields[catalog.UpdateVolumeRequestContent](append(slices.Clone(volumeForceSend), config.ForceSendFields...), "Owner"),
	}

	items := strings.Split(id, ".")
	if len(items) == 0 {
		return "", nil, fmt.Errorf("unexpected id=%#v", id)
	}
	nameFromID := items[len(items)-1]

	// The parents come from config: a renamed catalog or schema moves the volume with it,
	// leaving id stale (see followsParentRename).
	updateRequest.Name = config.CatalogName + "." + config.SchemaName + "." + nameFromID

	if entry.Changes.OnlyParentRenamed() {
		// UC already moved the volume with its parent; only the ID changes.
		remote, ok := entry.RemoteState.(*catalog.VolumeInfo)
		if !ok {
			return "", nil, fmt.Errorf("internal error: unexpected remote state type %T", entry.RemoteState)
		}
		config.VolumeId = remote.VolumeId
		return updateRequest.Name, nil, nil
	}

	if config.Name != nameFromID {
		updateRequest.NewName = config.Name
	}

	response, err := r.client.Volumes.Update(ctx, updateRequest)
	if err != nil || response == nil {
		return "", nil, err
	}

	config.VolumeId = response.VolumeId
	return response.FullName, response, nil
}

// IDFromState returns the full name, which is the ID.
func (*ResourceVolume) IDFromState(state *VolumeState) string {
	return state.CatalogName + "." + state.SchemaName + "." + state.Name
}

// InternalID returns the volume_id.
func (*ResourceVolume) InternalID(state *VolumeState) string {
	return state.VolumeId
}

func (r *ResourceVolume) DoDelete(ctx context.Context, id string, _ *VolumeState) error {
	return r.client.Volumes.DeleteByName(ctx, id)
}

func getNameFromID(id string) (string, error) {
	items := strings.Split(id, ".")
	if len(items) == 0 {
		return "", fmt.Errorf("unexpected id=%#v", id)
	}
	return items[len(items)-1], nil
}
