package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	// Echo back every field create accepts: a dropped one makes the next plan see a
	// phantom change.
	catalogInfo := catalog.CatalogInfo{
		Name:                      createRequest.Name,
		Comment:                   createRequest.Comment,
		ConnectionName:            createRequest.ConnectionName,
		CustomMaxRetentionHours:   createRequest.CustomMaxRetentionHours,
		ManagedEncryptionSettings: createRequest.ManagedEncryptionSettings,
		StorageRoot:               createRequest.StorageRoot,
		ProviderName:              createRequest.ProviderName,
		ShareName:                 createRequest.ShareName,
		Options:                   createRequest.Options,
		Properties:                createRequest.Properties,
		FullName:                  createRequest.Name,
		CreatedAt:                 nowMilli(),
		CreatedBy:                 s.CurrentUser().UserName,
		UpdatedBy:                 s.CurrentUser().UserName,
		MetastoreId:               nextUUID(),
		Owner:                     s.CurrentUser().UserName,
		CatalogType:               catalog.CatalogTypeManagedCatalog,
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
	if updateRequest.CustomMaxRetentionHours != 0 {
		existing.CustomMaxRetentionHours = updateRequest.CustomMaxRetentionHours
	}
	if updateRequest.ManagedEncryptionSettings != nil {
		existing.ManagedEncryptionSettings = updateRequest.ManagedEncryptionSettings
	}
	if updateRequest.Options != nil {
		existing.Options = updateRequest.Options
	}
	if updateRequest.Properties != nil {
		existing.Properties = updateRequest.Properties
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

	existing.UpdatedAt = nowMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Catalogs[name] = existing
	return Response{
		Body: existing,
	}
}
