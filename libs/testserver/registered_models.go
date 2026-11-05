package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"

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

	// The update API ignores aliases; they are set through the alias endpoints.
	delete(fields, "aliases")
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

// RegisteredModelsGet mirrors UC, which lists a model's aliases only when include_aliases is set.
func (s *FakeWorkspace) RegisteredModelsGet(req Request, fullName string) Response {
	defer s.LockUnlock()()

	model, ok := s.RegisteredModels[fullName]
	if !ok {
		return registeredModelNotFound(fullName)
	}
	if req.URL.Query().Get("include_aliases") != "true" {
		model.Aliases = nil
	}
	return Response{
		Body: model,
	}
}

func (s *FakeWorkspace) RegisteredModelsDelete(fullName string) Response {
	defer s.LockUnlock()()

	if _, ok := s.RegisteredModels[fullName]; !ok {
		return registeredModelNotFound(fullName)
	}
	delete(s.RegisteredModels, fullName)
	// The model's versions are deleted with it, so a model created later under the same name starts with none.
	delete(s.RegisteredModelVersions, fullName)
	return Response{}
}

// RegisteredModelsCreateVersion registers the model's next version. The MLflow client calls this endpoint of the
// MLflow Unity Catalog registry when registering a model; tests call it to give aliases a version to point at.
func (s *FakeWorkspace) RegisteredModelsCreateVersion(req Request) Response {
	defer s.LockUnlock()()

	var request struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}
	if _, ok := s.RegisteredModels[request.Name]; !ok {
		return registeredModelNotFound(request.Name)
	}

	s.RegisteredModelVersions[request.Name]++
	return Response{
		Body: map[string]any{
			"model_version": map[string]string{
				"name":    request.Name,
				"version": strconv.Itoa(s.RegisteredModelVersions[request.Name]),
			},
		},
	}
}

func (s *FakeWorkspace) RegisteredModelsSetAlias(req Request, fullName, aliasName string) Response {
	defer s.LockUnlock()()

	model, ok := s.RegisteredModels[fullName]
	if !ok {
		return registeredModelNotFound(fullName)
	}

	var request catalog.SetRegisteredModelAliasRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// An alias can only point at a registered version.
	if request.VersionNum < 1 || request.VersionNum > s.RegisteredModelVersions[fullName] {
		return Response{
			StatusCode: http.StatusNotFound,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Model version %d of registered model %s does not exist.", request.VersionNum, fullName),
			},
		}
	}

	alias := catalog.RegisteredModelAlias{
		AliasName:   aliasName,
		VersionNum:  request.VersionNum,
		Id:          nextUUID(),
		CatalogName: model.CatalogName,
		SchemaName:  model.SchemaName,
		ModelName:   model.Name,
	}

	// Copy rather than edit the stored slice in place: a response being written may still reference it.
	aliases := slices.Clone(model.Aliases)
	if i := slices.IndexFunc(aliases, func(a catalog.RegisteredModelAlias) bool { return a.AliasName == aliasName }); i >= 0 {
		aliases[i] = alias
	} else {
		aliases = append(aliases, alias)
	}
	model.Aliases = aliases
	s.RegisteredModels[fullName] = model
	return Response{
		Body: alias,
	}
}

func (s *FakeWorkspace) RegisteredModelsDeleteAlias(fullName, aliasName string) Response {
	defer s.LockUnlock()()

	model, ok := s.RegisteredModels[fullName]
	if !ok {
		return registeredModelNotFound(fullName)
	}

	model.Aliases = slices.DeleteFunc(slices.Clone(model.Aliases), func(a catalog.RegisteredModelAlias) bool {
		return a.AliasName == aliasName
	})
	s.RegisteredModels[fullName] = model
	return Response{}
}

func registeredModelNotFound(fullName string) Response {
	return Response{
		StatusCode: http.StatusNotFound,
		Body: map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    fmt.Sprintf("Registered model '%s' does not exist.", fullName),
		},
	}
}
