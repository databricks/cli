package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dario.cat/mergo"
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

func (s *FakeWorkspace) VolumesUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Volumes[name]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	var volume catalog.VolumeInfo

	if err := json.Unmarshal(req.Body, &volume); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	err := mergo.Merge(&existing, volume, mergo.WithOverride)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("mergo error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	s.Volumes[name] = existing
	return Response{
		Body: existing,
	}
}
