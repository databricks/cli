package resourcemutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type captureUCDependencies struct{}

// If a user defines a UC schema in the bundle, they can refer to it in SDP pipelines,
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
func CaptureUCDependencies() bundle.Mutator {
	return &captureUCDependencies{}
}

func (m *captureUCDependencies) Name() string {
	return "CaptureUCDependencies"
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

// resolveSchema returns the explicit schema reference if the given catalogName
// and schemaName match a schema defined in the bundle. Otherwise returns schemaName
// unchanged. Must be called before resolveCatalog on the same resource since
// findSchema needs the original (unmutated) catalogName.
func resolveSchema(b *bundle.Bundle, catalogName, schemaName string) string {
	k, s := findSchema(b, catalogName, schemaName)
	if s != nil {
		return schemaNameRef(k)
	}
	return schemaName
}

// resolveCatalog returns the explicit catalog reference if catalogName matches
// a catalog defined in the bundle. Otherwise returns catalogName unchanged.
func resolveCatalog(b *bundle.Bundle, catalogName string) string {
	k, c := findCatalog(b, catalogName)
	if c != nil {
		return catalogNameRef(k)
	}
	return catalogName
}

func (m *captureUCDependencies) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Resolve resources that depend on schemas before resolving schemas themselves.
	// The schema resolution below modifies schema.CatalogName, and findSchema
	// (used by resolveSchema) matches against the original schema.CatalogName value.
	for _, v := range b.Config.Resources.Volumes {
		if v == nil {
			continue
		}
		v.SchemaName = resolveSchema(b, v.CatalogName, v.SchemaName)
		v.CatalogName = resolveCatalog(b, v.CatalogName)
	}
	for _, rm := range b.Config.Resources.RegisteredModels {
		if rm == nil {
			continue
		}
		rm.SchemaName = resolveSchema(b, rm.CatalogName, rm.SchemaName)
		rm.CatalogName = resolveCatalog(b, rm.CatalogName)
	}
	for _, p := range b.Config.Resources.Pipelines {
		if p == nil {
			continue
		}
		// "schema" and "target" have the same semantics in the SDP API but are mutually
		// exclusive i.e. only one can be set at a time.
		p.Schema = resolveSchema(b, p.Catalog, p.Schema)
		p.Target = resolveSchema(b, p.Catalog, p.Target)
		p.Catalog = resolveCatalog(b, p.Catalog)
	}
	for _, qm := range b.Config.Resources.QualityMonitors {
		if qm == nil || qm.OutputSchemaName == "" {
			continue
		}
		// OutputSchemaName is a compound "catalog.schema" string.
		parts := strings.SplitN(qm.OutputSchemaName, ".", 2)
		if len(parts) != 2 {
			continue
		}
		catalogName, schemaName := parts[0], parts[1]
		resolved := resolveCatalog(b, catalogName) + "." + resolveSchema(b, catalogName, schemaName)
		if resolved != qm.OutputSchemaName {
			qm.OutputSchemaName = resolved
		}
	}
	for _, mse := range b.Config.Resources.ModelServingEndpoints {
		if mse == nil {
			continue
		}
		if mse.AiGateway != nil && mse.AiGateway.InferenceTableConfig != nil {
			itc := mse.AiGateway.InferenceTableConfig
			itc.SchemaName = resolveSchema(b, itc.CatalogName, itc.SchemaName)
			itc.CatalogName = resolveCatalog(b, itc.CatalogName)
		}
		// AutoCaptureConfig is deprecated but still in use.
		if mse.Config != nil && mse.Config.AutoCaptureConfig != nil {
			acc := mse.Config.AutoCaptureConfig
			acc.SchemaName = resolveSchema(b, acc.CatalogName, acc.SchemaName)
			acc.CatalogName = resolveCatalog(b, acc.CatalogName)
		}
	}

	// Schemas are resolved last because the schema catalog resolution modifies
	// schema.CatalogName, and findSchema (used by resolveSchema above) matches
	// against the original schema.CatalogName value.
	for _, s := range b.Config.Resources.Schemas {
		if s == nil {
			continue
		}
		s.CatalogName = resolveCatalog(b, s.CatalogName)
	}
	return nil
}
