package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (s *FakeWorkspace) CatalogsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest catalog.CreateCatalog
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	catalogInfo := catalog.CatalogInfo{
		Name:         createRequest.Name,
		Comment:      createRequest.Comment,
		StorageRoot:  createRequest.StorageRoot,
		ProviderName: createRequest.ProviderName,
		ShareName:    createRequest.ShareName,
		Options:      createRequest.Options,
		Properties:   createRequest.Properties,
		FullName:     createRequest.Name,
		CreatedAt:    time.Now().UnixMilli(),
		CreatedBy:    s.CurrentUser().UserName,
		UpdatedAt:    time.Now().UnixMilli(),
		UpdatedBy:    s.CurrentUser().UserName,
		MetastoreId:  nextUUID(),
		Owner:        s.CurrentUser().UserName,
		CatalogType:  catalog.CatalogTypeManagedCatalog,
	}

	s.Catalogs[createRequest.Name] = catalogInfo
	return Response{
		Body: catalogInfo,
	}
}

func (s *FakeWorkspace) CatalogsUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Catalogs[name]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("catalog %s not found", name),
		}
	}

	var updateRequest catalog.UpdateCatalog
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Update only the fields that can be updated
	if updateRequest.Comment != "" {
		existing.Comment = updateRequest.Comment
	}
	if updateRequest.Owner != "" {
		existing.Owner = updateRequest.Owner
	}
	if updateRequest.NewName != "" {
		existing.Name = updateRequest.NewName
		existing.FullName = updateRequest.NewName

		// Delete the old entry and create with new name
		delete(s.Catalogs, name)
		name = updateRequest.NewName
	}

	existing.UpdatedAt = time.Now().UnixMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Catalogs[name] = existing
	return Response{
		Body: existing,
	}
}
