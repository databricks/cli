package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

const testMetastoreName = "deco-uc-prod-isolated-aws-us-east-1"

// schemaNameManagedDefaults is the schema name the backend-default drift test uses
// to opt into UC's managed-property simulation. Scoping the injection to this name
// keeps unrelated schema tests free of the property, which terraform would otherwise
// report as drift on redeploy.
const schemaNameManagedDefaults = "schema_managed_defaults"

// ucInvalidNameResponse mirrors Unity Catalog's rejection of a name it will not accept, naming the RPC
// and the fully-qualified field the way the backend does.
func ucInvalidNameResponse(rpc, field, name string) Response {
	return Response{
		StatusCode: http.StatusBadRequest,
		Body: map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message": fmt.Sprintf("Invalid input: RPC %s Field %s: name %q is not a valid name. "+
				"Valid names cannot contain spaces, periods, forward slashes, or control characters.", rpc, field, name),
		},
	}
}

func (s *FakeWorkspace) SchemasCreate(req Request) Response {
	defer s.LockUnlock()()

	var schema catalog.SchemaInfo

	if err := json.Unmarshal(req.Body, &schema); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Unity Catalog validates both name parts before anything else, so an empty one is refused rather
	// than stored -- the fake server accepting it is what let a cleared name read OK locally while a
	// workspace refused it.
	for field, value := range map[string]string{"catalog_name": schema.CatalogName, "name": schema.Name} {
		if value == "" {
			return ucInvalidNameResponse("CreateSchema", "managedcatalog.SchemaInfo."+field, value)
		}
	}

	// UC normalizes schema names to lowercase.
	schema.Name = strings.ToLower(schema.Name)
	schema.FullName = schema.CatalogName + "." + schema.Name
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
	s.Schemas[schema.FullName] = schema

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

	var schemaUpdate catalog.SchemaInfo

	if err := json.Unmarshal(req.Body, &schemaUpdate); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, schemaUpdate, fields)

	existing.UpdatedAt = nowMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Schemas[name] = existing

	return Response{
		Body: existing,
	}
}
