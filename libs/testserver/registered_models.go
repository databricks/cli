package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	registeredmodels "github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	"github.com/databricks/cli/libs/utils"
)

// modelNameBrowseOnly scopes the computed browse_only flag to the browse_only
// drift test, keeping unrelated tests free of it.
const modelNameBrowseOnly = "model_browse_only"

func (s *FakeWorkspace) RegisteredModelsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest registeredmodels.CreateRegisteredModelRequest
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Build full name from catalog.schema.name
	fullName := utils.Deref(createRequest.CatalogName) + "." + utils.Deref(createRequest.SchemaName) + "." + utils.Deref(createRequest.Name)

	registeredModel := registeredmodels.RegisteredModelInfo{
		CatalogName:     createRequest.CatalogName,
		Comment:         createRequest.Comment,
		Name:            createRequest.Name,
		SchemaName:      createRequest.SchemaName,
		StorageLocation: createRequest.StorageLocation,
		FullName:        utils.NonZero(fullName),
		CreatedAt:       utils.NonZero(nowMilli()),
		CreatedBy:       s.CurrentUser().UserName,
		UpdatedBy:       s.CurrentUser().UserName,
		MetastoreId:     utils.NonZero(nextUUID()),
		Owner:           s.CurrentUser().UserName,
	}
	registeredModel.UpdatedAt = registeredModel.CreatedAt
	if utils.Deref(createRequest.Name) == modelNameBrowseOnly {
		// Mirror UC, which computes browse_only and echoes it on GET.
		registeredModel.BrowseOnly = utils.NonZero(true)
	}

	s.RegisteredModels[fullName] = registeredModel
	return Response{
		Body: registeredModel,
	}
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

	var updateRequest registeredmodels.UpdateRegisteredModelRequest
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, updateRequest, fields)

	if utils.Deref(updateRequest.NewName) != "" {
		existing.Name = updateRequest.NewName

		// Delete the old entry and set full name to the new name
		delete(s.RegisteredModels, fullName)
		fullName = utils.Deref(existing.CatalogName) + "." + utils.Deref(existing.SchemaName) + "." + utils.Deref(updateRequest.NewName)
	}

	existing.UpdatedAt = utils.NonZero(nowMilli())
	s.RegisteredModels[fullName] = existing
	return Response{
		Body: existing,
	}
}
