package testserver

import (
	"encoding/json"
	"fmt"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"net/http"
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

	return Response{
		Body: volume,
	}
}
