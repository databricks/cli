package testserver

import (
	"encoding/json"
	"fmt"
	volumes "github.com/databricks/cli/libs/sdktypes/uc/volumes/v1"
	workspaceapi "github.com/databricks/cli/libs/workspaceapi"
	"net/http"
	"strings"
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
	volume.SchemaName = nonZero(strings.ToLower(val(volume.SchemaName)))
	volume.Name = nonZero(strings.ToLower(val(volume.Name)))
	volume.FullName = nonZero(val(volume.CatalogName) + "." + val(volume.SchemaName) + "." + val(volume.Name))

	if val(volume.StorageLocation) != "" && volume.VolumeType != volumes.VolumeType_External {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "CreateVolume storage_location can not be provided.",
			},
		}
	}

	volume.VolumeId = nonZero(nextUUID())

	if val(volume.StorageLocation) != "" {
		// Strip trailing slash to mimic UC API normalization behavior.
		volume.StorageLocation = nonZero(strings.TrimRight(val(volume.StorageLocation), "/"))
	} else {
		volume.StorageLocation = nonZero(fmt.Sprintf("s3://%s/metastore/%s/volumes/%s", testMetastoreName, TestMetastore.MetastoreId, volume.VolumeId))
	}

	volume.CreatedAt = nonZero(nowMilli())
	volume.UpdatedAt = volume.CreatedAt
	volume.CreatedBy = s.CurrentUser().UserName
	volume.Owner = s.CurrentUser().UserName

	defer s.LockUnlock()()

	s.Volumes[val(volume.FullName)] = volume

	// Register the volume path as a directory so that fs cp works without
	// requiring an explicit mkdir.
	volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", volume.CatalogName, volume.SchemaName, volume.Name)
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

	if val(request.NewName) != "" {
		delete(s.Volumes, fullname)
		existing.Name = request.NewName
		existing.FullName = nonZero(val(existing.CatalogName) + "." + val(existing.SchemaName) + "." + val(request.NewName))
	}

	existing.UpdatedAt = nonZero(nowMilli())
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Volumes[val(existing.FullName)] = existing
	return Response{
		Body: existing,
	}
}
