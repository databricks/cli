package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	volumes "github.com/databricks/cli/libs/sdktypes/uc/volumes/v1"
	"github.com/databricks/cli/libs/utils"
	workspaceapi "github.com/databricks/cli/libs/workspaceapi"
)

func (s *FakeWorkspace) VolumesCreate(req Request) Response {
	var volume volumes.VolumeInfo

	if err := json.Unmarshal(req.Body, &volume); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// UC normalizes schema and volume names to lowercase.
	volume.SchemaName = utils.NonZero(strings.ToLower(utils.Deref(volume.SchemaName)))
	volume.Name = utils.NonZero(strings.ToLower(utils.Deref(volume.Name)))
	volume.FullName = utils.NonZero(utils.Deref(volume.CatalogName) + "." + utils.Deref(volume.SchemaName) + "." + utils.Deref(volume.Name))

	if utils.Deref(volume.StorageLocation) != "" && volume.VolumeType != volumes.VolumeType_External {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "CreateVolume storage_location can not be provided.",
			},
		}
	}

	volume.VolumeId = utils.NonZero(nextUUID())

	if utils.Deref(volume.StorageLocation) != "" {
		// Strip trailing slash to mimic UC API normalization behavior.
		volume.StorageLocation = utils.NonZero(strings.TrimRight(utils.Deref(volume.StorageLocation), "/"))
	} else {
		volume.StorageLocation = utils.NonZero(fmt.Sprintf("s3://%s/metastore/%s/volumes/%s", testMetastoreName, utils.Deref(TestMetastore.MetastoreId), utils.Deref(volume.VolumeId)))
	}

	volume.CreatedAt = utils.NonZero(nowMilli())
	volume.UpdatedAt = volume.CreatedAt
	volume.CreatedBy = s.CurrentUser().UserName
	volume.Owner = s.CurrentUser().UserName

	defer s.LockUnlock()()

	s.Volumes[utils.Deref(volume.FullName)] = volume

	// Register the volume path as a directory so that fs cp works without
	// requiring an explicit mkdir.
	volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", utils.Deref(volume.CatalogName), utils.Deref(volume.SchemaName), utils.Deref(volume.Name))
	s.directories[volumePath] = workspaceapi.ObjectInfo{
		ObjectType: "DIRECTORY",
		Path:       volumePath,
	}

	return Response{
		Body: volume,
	}
}

func (s *FakeWorkspace) VolumesUpdate(req Request, fullname string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Volumes[fullname]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	fields, errResponse := parseUCUpdate(req.Body, "UpdateVolume")
	if errResponse != nil {
		return *errResponse
	}

	var request volumes.UpdateVolumeRequest

	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, request, fields)

	if utils.Deref(request.NewName) != "" {
		delete(s.Volumes, fullname)
		existing.Name = request.NewName
		existing.FullName = utils.NonZero(utils.Deref(existing.CatalogName) + "." + utils.Deref(existing.SchemaName) + "." + utils.Deref(request.NewName))
	}

	existing.UpdatedAt = utils.NonZero(nowMilli())
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Volumes[utils.Deref(existing.FullName)] = existing
	return Response{
		Body: existing,
	}
}
