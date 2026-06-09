package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dario.cat/mergo"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

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
		Value:             catalog.EnablePredictiveOptimizationEnable,
	}
	schema.EnablePredictiveOptimization = catalog.EnablePredictiveOptimizationInherit
	schema.MetastoreId = TestMetastore.MetastoreId
	schema.Owner = s.CurrentUser().UserName
	schema.SchemaId = nextUUID()
	if schema.Properties == nil && schema.Name == schemaNameManagedDefaults {
		// Mirror UC behavior: managed system defaults are populated when the user
		// doesn't specify any properties. Required to cover backend-default drift.
		schema.Properties = map[string]string{
			"unity.catalog.managed.delta.defaults.delta.enableRowTracking": "true",
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

	var schemaUpdate catalog.SchemaInfo

	if err := json.Unmarshal(req.Body, &schemaUpdate); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	err := mergo.Merge(&existing, schemaUpdate, mergo.WithOverride)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("mergo error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	existing.UpdatedAt = nowMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.Schemas[name] = existing

	return Response{
		Body: existing,
	}
}
