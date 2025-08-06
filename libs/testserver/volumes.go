package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (s *FakeWorkspace) VolumesCreate(req Request) Response {
	var volume catalog.VolumeInfo

	if err := json.Unmarshal(req.Body, &volume); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	volume.FullName = volume.CatalogName + "." + volume.SchemaName + "." + volume.Name

	defer s.LockUnlock()()

	s.Volumes[volume.FullName] = volume
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

	var request catalog.UpdateVolumeRequestContent

	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if request.Comment != "" {
		existing.Comment = request.Comment
	}

	if request.Owner != "" {
		existing.Owner = request.Owner
	}

	if request.NewName != "" {
		delete(s.Volumes, fullname)
		existing.Name = request.NewName
		existing.FullName = existing.CatalogName + "." + existing.SchemaName + "." + request.NewName
	}

	s.Volumes[existing.FullName] = existing
	return Response{
		Body: existing,
	}
}
