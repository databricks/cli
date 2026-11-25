package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerRoutes("volumes", addVolumesRoutes)
}

func addVolumesRoutes(server *Server) {
	// Volumes: looked up case-insensitively, see volumeKey.

	server.Handle("GET", "/api/2.1/unity-catalog/volumes/{full_name}", func(req Request) any {
		return MapGetUC(req.Workspace, req.Workspace.Volumes, volumeKey(req.Vars["full_name"]), "Volume")
	})

	server.Handle("POST", "/api/2.1/unity-catalog/volumes", func(req Request) any {
		return req.Workspace.VolumesCreate(req)
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/volumes/{full_name}", func(req Request) any {
		return req.Workspace.VolumesUpdate(req, volumeKey(req.Vars["full_name"]))
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/volumes/{full_name}", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Volumes, volumeKey(req.Vars["full_name"]))
	})
}

func (s *FakeWorkspace) VolumesCreate(req Request) Response {
	var volume catalog.VolumeInfo

	if err := json.Unmarshal(req.Body, &volume); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// UC normalizes schema and volume names to lowercase.
	volume.SchemaName = strings.ToLower(volume.SchemaName)
	volume.Name = strings.ToLower(volume.Name)
	volume.FullName = volume.CatalogName + "." + volume.SchemaName + "." + volume.Name

	if volume.StorageLocation != "" && volume.VolumeType != catalog.VolumeTypeExternal {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "CreateVolume storage_location can not be provided.",
			},
		}
	}

	volume.VolumeId = nextUUID()

	if volume.StorageLocation != "" {
		// Strip trailing slash to mimic UC API normalization behavior.
		volume.StorageLocation = strings.TrimRight(volume.StorageLocation, "/")
	} else {
		volume.StorageLocation = fmt.Sprintf("s3://%s/metastore/%s/volumes/%s", testMetastoreName, TestMetastore.MetastoreId, volume.VolumeId)
	}

	volume.CreatedAt = nowMilli()
	volume.UpdatedAt = volume.CreatedAt
	volume.CreatedBy = s.CurrentUser().UserName
	volume.Owner = s.CurrentUser().UserName

	defer s.LockUnlock()()

	s.Volumes[volumeKey(volume.FullName)] = volume

	// Register the volume path as a directory so that fs cp works without
	// requiring an explicit mkdir.
	volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", volume.CatalogName, volume.SchemaName, volume.Name)
	s.directories[volumePath] = workspace.ObjectInfo{
		ObjectType: "DIRECTORY",
		Path:       volumePath,
	}

	return Response{
		Body: volume,
	}
}

// volumeKey is the key a volume is stored under. UC looks a full name up case-insensitively
// (checked on AWS: a volume re-keyed as main.s.MyVolume is still read at that path).
func volumeKey(fullName string) string {
	return strings.ToLower(fullName)
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

	var request catalog.UpdateVolumeRequestContent

	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, request, fields)

	if request.NewName != "" {
		delete(s.Volumes, fullname)
		// UC lowercases the new name, as on create.
		existing.Name = strings.ToLower(request.NewName)
		existing.FullName = existing.CatalogName + "." + existing.SchemaName + "." + existing.Name
	}

	existing.UpdatedAt = nowMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Volumes[volumeKey(existing.FullName)] = existing
	return Response{
		Body: existing,
	}
}
