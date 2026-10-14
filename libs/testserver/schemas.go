package testserver

import (
	"encoding/json"
	"fmt"
	schemas "github.com/databricks/cli/libs/sdktypes/uc/schemas/v1"
	"net/http"
	"strings"
)

const testMetastoreName = "deco-uc-prod-isolated-aws-us-east-1"

// schemaNameManagedDefaults is the schema name the backend-default drift test uses
// to opt into UC's managed-property simulation. Scoping the injection to this name
// keeps unrelated schema tests free of the property, which terraform would otherwise
// report as drift on redeploy.
const schemaNameManagedDefaults = "schema_managed_defaults"

func (s *FakeWorkspace) SchemasCreate(req Request) Response {
	defer s.LockUnlock()()

	var schema schemas.SchemaInfo

	if err := json.Unmarshal(req.Body, &schema); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// UC normalizes schema names to lowercase.
	schema.Name = nonZero(strings.ToLower(val(schema.Name)))
	// Strip trailing slash to mimic UC API normalization behavior (see volumes.go).
	schema.StorageRoot = nonZero(strings.TrimRight(val(schema.StorageRoot), "/"))
	schema.FullName = nonZero(val(schema.CatalogName) + "." + val(schema.Name))
	schema.BrowseOnly = new(false)
	schema.CatalogType = "MANAGED_CATALOG"
	schema.CreatedAt = nonZero(nowMilli())
	schema.UpdatedAt = schema.CreatedAt
	schema.CreatedBy = s.CurrentUser().UserName
	schema.UpdatedBy = s.CurrentUser().UserName
	schema.EffectivePredictiveOptimizationFlag = &schemas.EffectivePredictiveOptimizationFlag{
		InheritedFromName: nonZero(testMetastoreName),
		InheritedFromType: new("METASTORE"),
		// Mirror the real test metastore, which inherits ENABLE, so a single
		// golden stays valid for both local and cloud runs.
		Value: new("ENABLE"),
	}
	schema.EnablePredictiveOptimization = new("INHERIT")
	schema.MetastoreId = TestMetastore.MetastoreId
	schema.Owner = s.CurrentUser().UserName
	schema.SchemaId = nonZero(nextUUID())
	if schema.Properties == nil && val(schema.Name) == schemaNameManagedDefaults {
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
	s.Schemas[val(schema.FullName)] = schema

	return Response{
		Body: schema,
	}
}

func (s *FakeWorkspace) SchemasUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Schemas[name]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	fields, errResponse := parseUCUpdate(req.Body, "UpdateSchema")
	if errResponse != nil {
		return *errResponse
	}

	var schemaUpdate schemas.SchemaInfo

	if err := json.Unmarshal(req.Body, &schemaUpdate); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, schemaUpdate, fields)

	existing.UpdatedAt = nonZero(nowMilli())
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Schemas[name] = existing

	return Response{
		Body: existing,
	}
}
