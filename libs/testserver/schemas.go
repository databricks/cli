package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerRoutes("schemas", addSchemasRoutes)
	registerStore("schemas", newSchemasStore)
}

type schemasStore struct {
	Schemas map[string]catalog.SchemaInfo
}

func newSchemasStore(*FakeWorkspace) *schemasStore {
	return &schemasStore{
		Schemas: map[string]catalog.SchemaInfo{},
	}
}

func (s *FakeWorkspace) schemas() *schemasStore {
	return stateOf[schemasStore](s)
}

func addSchemasRoutes(server *Server) {
	// Schemas:

	server.Handle("GET", "/api/2.1/unity-catalog/schemas/{full_name}", func(req Request) any {
		return MapGetUC(req.Workspace, req.Workspace.schemas().Schemas, req.Vars["full_name"], "Schema")
	})

	server.Handle("POST", "/api/2.1/unity-catalog/schemas", func(req Request) any {
		return req.Workspace.SchemasCreate(req)
	})

	server.Handle("PATCH", "/api/2.1/unity-catalog/schemas/{full_name}", func(req Request) any {
		return req.Workspace.SchemasUpdate(req, req.Vars["full_name"])
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/schemas/{full_name}", func(req Request) any {
		return req.Workspace.SchemasDelete(req, req.Vars["full_name"])
	})
}

const testMetastoreName = "deco-uc-prod-isolated-aws-us-east-1"

// schemaNameManagedDefaults is the schema name the backend-default drift test uses
// to opt into UC's managed-property simulation. Scoping the injection to this name
// keeps unrelated schema tests free of the property, which terraform would otherwise
// report as drift on redeploy.
const schemaNameManagedDefaults = "schema_managed_defaults"

func (s *FakeWorkspace) SchemasCreate(req Request) Response {
	defer s.LockUnlock()()

	var schema catalog.SchemaInfo

	if err := json.Unmarshal(req.Body, &schema); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// UC normalizes schema names to lowercase.
	schema.Name = strings.ToLower(schema.Name)
	// Strip trailing slash to mimic UC API normalization behavior (see volumes.go).
	schema.StorageRoot = strings.TrimRight(schema.StorageRoot, "/")
	schema.FullName = schema.CatalogName + "." + schema.Name

	if _, exists := s.schemas().Schemas[schema.FullName]; exists {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "SCHEMA_ALREADY_EXISTS",
				"message":    fmt.Sprintf("Schema '%s' already exists", schema.Name),
			},
		}
	}
	schema.ForceSendFields = []string{"BrowseOnly"}
	schema.CatalogType = "MANAGED_CATALOG"
	schema.CreatedAt = nowMilli()
	schema.UpdatedAt = schema.CreatedAt
	schema.CreatedBy = s.CurrentUser().UserName
	schema.UpdatedBy = s.CurrentUser().UserName
	schema.EffectivePredictiveOptimizationFlag = &catalog.EffectivePredictiveOptimizationFlag{
		InheritedFromName: testMetastoreName,
		InheritedFromType: catalog.EffectivePredictiveOptimizationFlagInheritedFromType("METASTORE"),
		// Mirror the real test metastore, which inherits ENABLE, so a single
		// golden stays valid for both local and cloud runs.
		Value: catalog.EnablePredictiveOptimizationEnable,
	}
	schema.EnablePredictiveOptimization = catalog.EnablePredictiveOptimizationInherit
	schema.MetastoreId = TestMetastore.MetastoreId
	schema.Owner = s.CurrentUser().UserName
	schema.SchemaId = nextUUID()
	if schema.Properties == nil && schema.Name == schemaNameManagedDefaults {
		// Mirror UC behavior: managed system defaults are populated when the user
		// doesn't specify any properties. Required to cover backend-default drift.
		schema.Properties = map[string]string{
			"unity.catalog.managed.delta.defaults.delta.enableRowTracking":                   "true",
			"unity.catalog.managed.iceberg.defaults.delta.feature.catalogManaged":            "true",
			"unity.catalog.managed.delta.defaults.defaultClusterByAuto":                      "true",
			"unity.catalog.managed.delta.defaults.delta.checkpointPolicy":                    "v2",
			"unity.catalog.managed.delta.defaults.delta.parquet.format.version":              "2.12.0",
			"unity.catalog.managed.delta.defaults.delta.parquet.format.version.afe.internal": "2.12.0",
			"unity.catalog.managed.delta.defaults.delta.feature.catalogManaged":              "supported",
		}
	}
	s.schemas().Schemas[schema.FullName] = schema

	return Response{
		Body: schema,
	}
}

func (s *FakeWorkspace) SchemasUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.schemas().Schemas[name]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	fields, errResponse := parseUCUpdate(req.Body, "UpdateSchema")
	if errResponse != nil {
		return *errResponse
	}

	var schemaUpdate catalog.UpdateSchema

	if err := json.Unmarshal(req.Body, &schemaUpdate); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, schemaUpdate, fields)

	if schemaUpdate.NewName != "" {
		delete(s.schemas().Schemas, name)
		oldName := existing.Name
		// UC normalizes schema names to lowercase.
		existing.Name = strings.ToLower(schemaUpdate.NewName)
		existing.FullName = existing.CatalogName + "." + existing.Name
		s.moveSchemaChildren(existing.CatalogName, oldName, existing.Name)
	}

	existing.UpdatedAt = nowMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.schemas().Schemas[existing.FullName] = existing

	return Response{
		Body: existing,
	}
}

// moveSchemaChildren mirrors UC, where the volumes, registered models and secrets of a renamed schema move with it.
func (s *FakeWorkspace) moveSchemaChildren(catalogName, oldSchemaName, newSchemaName string) {
	for key, volume := range s.volumes().Volumes {
		if volume.CatalogName != catalogName || volume.SchemaName != oldSchemaName {
			continue
		}
		delete(s.volumes().Volumes, key)
		volume.SchemaName = newSchemaName
		volume.FullName = catalogName + "." + newSchemaName + "." + volume.Name
		s.volumes().Volumes[volumeKey(volume.FullName)] = volume
	}
	for key, model := range s.registeredModels().RegisteredModels {
		if model.CatalogName != catalogName || model.SchemaName != oldSchemaName {
			continue
		}
		model.SchemaName = newSchemaName
		s.moveRegisteredModel(key, model)
	}
	for key, secret := range s.ucSecrets().UCSecrets {
		if secret.CatalogName != catalogName || secret.SchemaName != oldSchemaName {
			continue
		}
		delete(s.ucSecrets().UCSecrets, key)
		secret.SchemaName = newSchemaName
		secret.FullName = catalogName + "." + newSchemaName + "." + secret.Name
		s.ucSecrets().UCSecrets[secret.FullName] = secret
	}
}

// SchemasDelete mirrors UC, where deleting a schema with force deletes the volumes, registered models and secrets in it too.
func (s *FakeWorkspace) SchemasDelete(req Request, fullName string) Response {
	defer s.LockUnlock()()

	schema, ok := s.schemas().Schemas[fullName]
	if !ok {
		return Response{StatusCode: http.StatusNotFound}
	}
	delete(s.schemas().Schemas, fullName)
	if req.URL.Query().Get("force") == "true" {
		for key, volume := range s.volumes().Volumes {
			if volume.CatalogName == schema.CatalogName && volume.SchemaName == schema.Name {
				delete(s.volumes().Volumes, key)
			}
		}
		for key, model := range s.registeredModels().RegisteredModels {
			if model.CatalogName == schema.CatalogName && model.SchemaName == schema.Name {
				delete(s.registeredModels().RegisteredModels, key)
				delete(s.registeredModels().RegisteredModelVersions, key)
			}
		}
		for key, secret := range s.ucSecrets().UCSecrets {
			if secret.CatalogName == schema.CatalogName && secret.SchemaName == schema.Name {
				delete(s.ucSecrets().UCSecrets, key)
			}
		}
	}
	return Response{}
}
