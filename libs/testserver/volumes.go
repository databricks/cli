package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (*FakeWorkspace) VolumesCreate(req Request) Response {
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
