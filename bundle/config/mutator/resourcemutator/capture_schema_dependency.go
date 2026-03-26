package resourcemutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type captureSchemaDependency struct{}

// If a user defines a UC schema in the bundle, they can refer to it in DLT pipelines,
// UC Volumes, Registered Models, Quality Monitors, or Model Serving Endpoints using the
// `${resources.schemas.<schema_key>.name}` syntax. Using this syntax allows TF to capture
// the deploy time dependency this resource has on the schema and deploy changes to the
// schema before deploying the dependent resource.
//
// Similarly, if a user defines a UC catalog in the bundle, they can refer to it in UC schemas,
// UC Volumes, Registered Models, or Model Serving Endpoints using the
// `${resources.catalogs.<catalog_key>.name}` syntax. This captures the deploy time
// dependency the resource has on the catalog.
//
// This mutator translates any implicit catalog or schema references to the explicit syntax.
func CaptureSchemaDependency() bundle.Mutator {
	return &captureSchemaDependency{}
}

func (m *captureSchemaDependency) Name() string {
	return "CaptureSchemaDependency"
}

func schemaNameRef(key string) string {
	return fmt.Sprintf("${resources.schemas.%s.name}", key)
}

func catalogNameRef(key string) string {
	return fmt.Sprintf("${resources.catalogs.%s.name}", key)
}

func findSchema(b *bundle.Bundle, catalogName, schemaName string) (string, *resources.Schema) {
	if catalogName == "" || schemaName == "" {
		return "", nil
	}

	for k, s := range b.Config.Resources.Schemas {
		if s != nil && s.CatalogName == catalogName && s.Name == schemaName {
			return k, s
		}
	}
	return "", nil
}

func resolveVolume(v *resources.Volume, b *bundle.Bundle) {
	if v == nil {
		return
	}
	// Resolve schema first since findSchema needs the original v.CatalogName.
	schemaK, schema := findSchema(b, v.CatalogName, v.SchemaName)
	if schema != nil {
		v.SchemaName = schemaNameRef(schemaK)
	}

	catalogK, catalog := findCatalog(b, v.CatalogName)
	if catalog != nil {
		v.CatalogName = catalogNameRef(catalogK)
	}
}

func resolveRegisteredModel(rm *resources.RegisteredModel, b *bundle.Bundle) {
	if rm == nil {
		return
	}
	// Resolve schema first since findSchema needs the original rm.CatalogName.
	schemaK, schema := findSchema(b, rm.CatalogName, rm.SchemaName)
	if schema != nil {
		rm.SchemaName = schemaNameRef(schemaK)
	}

	catalogK, catalog := findCatalog(b, rm.CatalogName)
	if catalog != nil {
		rm.CatalogName = catalogNameRef(catalogK)
	}
}

func resolvePipelineSchema(p *resources.Pipeline, b *bundle.Bundle) {
	if p == nil {
		return
	}
	if p.Schema == "" {
		return
	}
	schemaK, schema := findSchema(b, p.Catalog, p.Schema)
	if schema == nil {
		return
	}

	p.Schema = schemaNameRef(schemaK)
}

func resolvePipelineTarget(p *resources.Pipeline, b *bundle.Bundle) {
	if p == nil {
		return
	}
	if p.Target == "" {
		return
	}
	schemaK, schema := findSchema(b, p.Catalog, p.Target)
	if schema == nil {
		return
	}
	p.Target = schemaNameRef(schemaK)
}

func findCatalog(b *bundle.Bundle, catalogName string) (string, *resources.Catalog) {
	if catalogName == "" {
		return "", nil
	}

	for k, c := range b.Config.Resources.Catalogs {
		if c != nil && c.Name == catalogName {
			return k, c
		}
	}
	return "", nil
}

// resolveQualityMonitor resolves the OutputSchemaName field which is a compound
// "catalog.schema" string.
func resolveQualityMonitor(qm *resources.QualityMonitor, b *bundle.Bundle) {
	if qm == nil {
		return
	}
	if qm.OutputSchemaName == "" {
		return
	}

	parts := strings.SplitN(qm.OutputSchemaName, ".", 2)
	if len(parts) != 2 {
		return
	}
	catalogName, schemaName := parts[0], parts[1]

	resolvedCatalog, resolvedSchema := catalogName, schemaName

	schemaK, schema := findSchema(b, catalogName, schemaName)
	if schema != nil {
		resolvedSchema = schemaNameRef(schemaK)
	}

	catalogK, catalog := findCatalog(b, catalogName)
	if catalog != nil {
		resolvedCatalog = catalogNameRef(catalogK)
	}

	if resolvedCatalog != catalogName || resolvedSchema != schemaName {
		qm.OutputSchemaName = resolvedCatalog + "." + resolvedSchema
	}
}

func resolveModelServingEndpoint(mse *resources.ModelServingEndpoint, b *bundle.Bundle) {
	if mse == nil {
		return
	}

	// Resolve AiGateway.InferenceTableConfig.
	if mse.AiGateway != nil && mse.AiGateway.InferenceTableConfig != nil {
		itc := mse.AiGateway.InferenceTableConfig

		schemaK, schema := findSchema(b, itc.CatalogName, itc.SchemaName)
		if schema != nil {
			itc.SchemaName = schemaNameRef(schemaK)
		}

		catalogK, catalog := findCatalog(b, itc.CatalogName)
		if catalog != nil {
			itc.CatalogName = catalogNameRef(catalogK)
		}
	}

	// Resolve Config.AutoCaptureConfig (deprecated but still in use).
	if mse.Config != nil && mse.Config.AutoCaptureConfig != nil {
		acc := mse.Config.AutoCaptureConfig

		schemaK, schema := findSchema(b, acc.CatalogName, acc.SchemaName)
		if schema != nil {
			acc.SchemaName = schemaNameRef(schemaK)
		}

		catalogK, catalog := findCatalog(b, acc.CatalogName)
		if catalog != nil {
			acc.CatalogName = catalogNameRef(catalogK)
		}
	}
}

func resolveSchema(s *resources.Schema, b *bundle.Bundle) {
	if s == nil {
		return
	}
	catalogK, catalog := findCatalog(b, s.CatalogName)
	if catalog == nil {
		return
	}

	s.CatalogName = catalogNameRef(catalogK)
}

func (m *captureSchemaDependency) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Resolve resources that depend on schemas before resolving schemas themselves.
	// resolveSchema modifies schema.CatalogName, and findSchema (used by the other
	// resolve functions) matches against the original schema.CatalogName value.
	for _, v := range b.Config.Resources.Volumes {
		resolveVolume(v, b)
	}
	for _, rm := range b.Config.Resources.RegisteredModels {
		resolveRegisteredModel(rm, b)
	}
	for _, p := range b.Config.Resources.Pipelines {
		// "schema" and "target" have the same semantics in the DLT API but are mutually
		// exclusive i.e. only one can be set at a time. If schema is set, the pipeline
		// is in direct publishing mode and can write tables to multiple schemas
		// (vs target which is limited to a single schema).
		resolvePipelineTarget(p, b)
		resolvePipelineSchema(p, b)
	}
	for _, qm := range b.Config.Resources.QualityMonitors {
		resolveQualityMonitor(qm, b)
	}
	for _, mse := range b.Config.Resources.ModelServingEndpoints {
		resolveModelServingEndpoint(mse, b)
	}
	for _, s := range b.Config.Resources.Schemas {
		resolveSchema(s, b)
	}
	return nil
}
