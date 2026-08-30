package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// modelNameBrowseOnly scopes the computed browse_only flag to the browse_only
// drift test, keeping unrelated tests free of it.
const modelNameBrowseOnly = "model_browse_only"

func (s *FakeWorkspace) RegisteredModelsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest catalog.CreateRegisteredModelRequest
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// UC requires all three parts of the name; without this the fake would store a model
	// under a key like "..name" that no read can address.
	for _, required := range []struct{ field, value string }{
		{"catalog_name", createRequest.CatalogName},
		{"schema_name", createRequest.SchemaName},
		{"name", createRequest.Name},
	} {
		if required.value == "" {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body: map[string]string{
					"error_code": "INVALID_PARAMETER_VALUE",
					"message":    "CreateRegisteredModel Missing required field: " + required.field,
				},
			}
		}
	}

	// Build full name from catalog.schema.name
	fullName := createRequest.CatalogName + "." + createRequest.SchemaName + "." + createRequest.Name

	registeredModel := catalog.RegisteredModelInfo{
		CatalogName:     createRequest.CatalogName,
		Comment:         createRequest.Comment,
		Name:            createRequest.Name,
		SchemaName:      createRequest.SchemaName,
		StorageLocation: createRequest.StorageLocation,
		FullName:        fullName,
		CreatedAt:       nowMilli(),
		CreatedBy:       s.CurrentUser().UserName,
		UpdatedBy:       s.CurrentUser().UserName,
		MetastoreId:     nextUUID(),
		Owner:           s.CurrentUser().UserName,
	}
	registeredModel.UpdatedAt = registeredModel.CreatedAt
	if createRequest.Name == modelNameBrowseOnly {
		// Mirror UC, which computes browse_only and echoes it on GET.
		registeredModel.BrowseOnly = true
	}

	s.RegisteredModels[fullName] = registeredModel
	return Response{
		Body: registeredModel,
	}
}

// RegisteredModelsGet honors include_aliases, which UC defaults to false: an alias belongs
// to a model version and is managed through its own API, so a plain GET does not echo the
// aliases a create or update was given. Without this the remote appeared to hold them and a
// config that sets aliases looked like it converged.
func (s *FakeWorkspace) RegisteredModelsGet(req Request, fullName string) Response {
	response := MapGetUC(s, s.RegisteredModels, fullName, "Registered Model")
	model, ok := response.Body.(catalog.RegisteredModelInfo)
	if !ok || req.URL.Query().Get("include_aliases") == "true" {
		return response
	}
	model.Aliases = nil
	response.Body = model
	return response
}

func (s *FakeWorkspace) RegisteredModelsUpdate(req Request, fullName string) Response {
	defer s.LockUnlock()()

	existing, ok := s.RegisteredModels[fullName]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("registered model %s not found", fullName),
		}
	}

	fields, errResponse := parseUpdateFields(req.Body)
	if errResponse != nil {
		return *errResponse
	}

	var updateRequest catalog.UpdateRegisteredModelRequest
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, updateRequest, fields)

	if updateRequest.NewName != "" {
		existing.Name = updateRequest.NewName

		// Delete the old entry and set full name to the new name
		delete(s.RegisteredModels, fullName)
		fullName = existing.CatalogName + "." + existing.SchemaName + "." + updateRequest.NewName
	}

	existing.UpdatedAt = nowMilli()
	s.RegisteredModels[fullName] = existing
	return Response{
		Body: existing,
	}
}
