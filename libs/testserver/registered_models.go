package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (s *FakeWorkspace) RegisteredModelsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest catalog.CreateRegisteredModelRequest
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
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
		CreatedAt:       time.Now().UnixMilli(),
		CreatedBy:       s.CurrentUser().UserName,
		UpdatedAt:       time.Now().UnixMilli(),
		UpdatedBy:       s.CurrentUser().UserName,
		MetastoreId:     nextUUID(),
		Owner:           s.CurrentUser().UserName,
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

	var updateRequest catalog.UpdateRegisteredModelRequest
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

		// Delete the old entry and set full name to the new name
		delete(s.RegisteredModels, fullName)
		fullName = existing.CatalogName + "." + existing.SchemaName + "." + updateRequest.NewName
	}

	existing.UpdatedAt = time.Now().UnixMilli()
	s.RegisteredModels[fullName] = existing
	return Response{
		Body: existing,
	}
}

func (s *FakeWorkspace) RegisteredModelsSetAlias(req Request, fullName, alias string) Response {
	defer s.LockUnlock()()

	existing, ok := s.RegisteredModels[fullName]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("registered model %s not found", fullName),
		}
	}

	var setRequest catalog.SetRegisteredModelAliasRequest
	if err := json.Unmarshal(req.Body, &setRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	newAlias := catalog.RegisteredModelAlias{
		AliasName:  alias,
		VersionNum: setRequest.VersionNum,
	}

	// Update existing alias or append new one.
	found := false
	for i, a := range existing.Aliases {
		if a.AliasName == alias {
			existing.Aliases[i] = newAlias
			found = true
			break
		}
	}
	if !found {
		existing.Aliases = append(existing.Aliases, newAlias)
	}

	s.RegisteredModels[fullName] = existing
	return Response{
		Body: newAlias,
	}
}

func (s *FakeWorkspace) RegisteredModelsDeleteAlias(fullName, alias string) Response {
	defer s.LockUnlock()()

	existing, ok := s.RegisteredModels[fullName]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("registered model %s not found", fullName),
		}
	}

	for i, a := range existing.Aliases {
		if a.AliasName == alias {
			existing.Aliases = append(existing.Aliases[:i], existing.Aliases[i+1:]...)
			break
		}
	}

	s.RegisteredModels[fullName] = existing
	return Response{}
}
