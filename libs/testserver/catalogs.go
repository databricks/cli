package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// catalogNameManagedDefaults scopes the managed-property injection to the
// backend-default drift test, keeping unrelated tests free of it. Mirrors
// schemaNameManagedDefaults.
const catalogNameManagedDefaults = "catalog_managed_defaults"

func (s *FakeWorkspace) CatalogsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest catalog.CreateCatalog
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// UC rejects an empty name; the fake would otherwise store a catalog under a key
	// nothing can look up. Message is UC's canned error, which names more than we check.
	if createRequest.Name == "" {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    `Invalid input: RPC CreateCatalog Field managedcatalog.CatalogInfo.name: name "" is not a valid name. Valid names cannot contain spaces, periods, forward slashes, or control characters.`,
			},
		}
	}

	if _, exists := s.Catalogs[createRequest.Name]; exists {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "CATALOG_ALREADY_EXISTS",
				"message":    fmt.Sprintf("Catalog '%s' already exists", createRequest.Name),
			},
		}
	}

	// Echo back every field create accepts: a dropped one makes the next plan see a
	// phantom change.
	catalogInfo := catalog.CatalogInfo{
		Name:                      createRequest.Name,
		Comment:                   createRequest.Comment,
		ConnectionName:            createRequest.ConnectionName,
		CustomMaxRetentionHours:   createRequest.CustomMaxRetentionHours,
		ManagedEncryptionSettings: createRequest.ManagedEncryptionSettings,
		// Strip trailing slash to mimic UC API normalization behavior (see volumes.go).
		StorageRoot:  strings.TrimRight(createRequest.StorageRoot, "/"),
		ProviderName: createRequest.ProviderName,
		ShareName:    createRequest.ShareName,
		Options:      createRequest.Options,
		Properties:   createRequest.Properties,
		FullName:     createRequest.Name,
		CreatedAt:    nowMilli(),
		CreatedBy:    s.CurrentUser().UserName,
		UpdatedBy:    s.CurrentUser().UserName,
		MetastoreId:  nextUUID(),
		Owner:        s.CurrentUser().UserName,
		CatalogType:  catalog.CatalogTypeManagedCatalog,
	}
	catalogInfo.UpdatedAt = catalogInfo.CreatedAt
	if catalogInfo.Properties == nil && createRequest.Name == catalogNameManagedDefaults {
		// Mirror UC: managed system defaults are populated when the user sets none.
		catalogInfo.Properties = map[string]string{
			"unity.catalog.managed.delta.defaults.delta.enableRowTracking":                   "true",
			"unity.catalog.managed.iceberg.defaults.delta.feature.catalogManaged":            "true",
			"unity.catalog.managed.delta.defaults.defaultClusterByAuto":                      "true",
			"unity.catalog.managed.delta.defaults.delta.checkpointPolicy":                    "v2",
			"unity.catalog.managed.delta.defaults.delta.parquet.format.version":              "2.12.0",
			"unity.catalog.managed.delta.defaults.delta.parquet.format.version.afe.internal": "2.12.0",
			"unity.catalog.managed.delta.defaults.delta.feature.catalogManaged":              "supported",
		}
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

	fields, errResponse := parseUCUpdate(req.Body, "UpdateCatalog")
	if errResponse != nil {
		return *errResponse
	}

	var updateRequest catalog.UpdateCatalog
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, updateRequest, fields)
	if updateRequest.NewName != "" {
		existing.Name = updateRequest.NewName
		existing.FullName = updateRequest.NewName

		// Delete the old entry and create with new name
		delete(s.Catalogs, name)
		s.moveCatalogChildren(name, updateRequest.NewName)
		name = updateRequest.NewName
	}

	existing.UpdatedAt = nowMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Catalogs[name] = existing
	return Response{
		Body: existing,
	}
}

// moveCatalogChildren mirrors UC, where the schemas, volumes, registered models and secrets of a renamed catalog move with it.
func (s *FakeWorkspace) moveCatalogChildren(oldCatalogName, newCatalogName string) {
	for key, schema := range s.Schemas {
		if schema.CatalogName != oldCatalogName {
			continue
		}
		delete(s.Schemas, key)
		schema.CatalogName = newCatalogName
		schema.FullName = newCatalogName + "." + schema.Name
		s.Schemas[schema.FullName] = schema
	}
	for key, volume := range s.Volumes {
		if volume.CatalogName != oldCatalogName {
			continue
		}
		delete(s.Volumes, key)
		volume.CatalogName = newCatalogName
		volume.FullName = newCatalogName + "." + volume.SchemaName + "." + volume.Name
		s.Volumes[volume.FullName] = volume
	}
	for key, model := range s.RegisteredModels {
		if model.CatalogName != oldCatalogName {
			continue
		}
		model.CatalogName = newCatalogName
		s.moveRegisteredModel(key, model)
	}
	for key, secret := range s.UCSecrets {
		if secret.CatalogName != oldCatalogName {
			continue
		}
		delete(s.UCSecrets, key)
		secret.CatalogName = newCatalogName
		secret.FullName = newCatalogName + "." + secret.SchemaName + "." + secret.Name
		s.UCSecrets[secret.FullName] = secret
	}
}

// CatalogsDelete mirrors UC, where deleting a catalog with force deletes its schemas, volumes, registered models and secrets too.
func (s *FakeWorkspace) CatalogsDelete(req Request, name string) Response {
	defer s.LockUnlock()()

	if _, ok := s.Catalogs[name]; !ok {
		return Response{StatusCode: http.StatusNotFound}
	}
	delete(s.Catalogs, name)
	if req.URL.Query().Get("force") == "true" {
		for key, schema := range s.Schemas {
			if schema.CatalogName == name {
				delete(s.Schemas, key)
			}
		}
		for key, volume := range s.Volumes {
			if volume.CatalogName == name {
				delete(s.Volumes, key)
			}
		}
		for key, model := range s.RegisteredModels {
			if model.CatalogName == name {
				delete(s.RegisteredModels, key)
				delete(s.RegisteredModelVersions, key)
			}
		}
		for key, secret := range s.UCSecrets {
			if secret.CatalogName == name {
				delete(s.UCSecrets, key)
			}
		}
	}
	return Response{}
}
