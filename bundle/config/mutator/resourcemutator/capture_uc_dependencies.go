package resourcemutator

import (
	"context"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type captureUCDependencies struct{}

// If a user defines a UC schema or catalog in the bundle and refers to it by its literal
// name from another resource (UC Volumes, SDP pipelines, Registered Models, Quality Monitors,
// Model Serving Endpoints, Vector Search Indexes, AI Gateway services, UC schemas), the dependent
// resource must be deployed after the schema or catalog.
//
// This mutator captures such implicit dependencies in lifecycle.depends_on. Name prefixes
// applied to schemas by presets are applied to these references by ApplyPresets.
func CaptureUCDependencies() bundle.Mutator {
	return &captureUCDependencies{}
}

func (m *captureUCDependencies) Name() string {
	return "CaptureUCDependencies"
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

// splitUCName splits a compound UC identifier into exactly n dot-separated
// components, returning false if it has a different number of components. A
// ${...} reference has dots too, so it pushes the component count past n and is
// rejected here.
func splitUCName(name string, n int) ([]string, bool) {
	parts := strings.Split(name, ".")
	if len(parts) != n {
		return nil, false
	}
	return parts, true
}

// schemaRefVisitor is called for every reference from a resource to a UC schema, given by
// the catalog and schema names. It returns the schema name to store back.
type schemaRefVisitor func(lc *resources.Lifecycle, catalogName, schemaName string) string

// visitCompoundSchemaRef visits the schema part of a "catalog.schema[.suffix]" name that has n components.
func visitCompoundSchemaRef(lc *resources.Lifecycle, name string, n int, visit schemaRefVisitor) string {
	parts, ok := splitUCName(name, n)
	if !ok {
		return name
	}
	parts[1] = visit(lc, parts[0], parts[1])
	return strings.Join(parts, ".")
}

// visitAiGatewayParent visits the schema in a `schemas/{catalog}.{schema}` parent. AI Gateway
// securables address their parent schema with this compound field rather than separate
// catalog/schema fields.
func visitAiGatewayParent(lc *resources.Lifecycle, parent string, visit schemaRefVisitor) string {
	rest, ok := strings.CutPrefix(parent, "schemas/")
	if !ok {
		return parent
	}
	return "schemas/" + visitCompoundSchemaRef(lc, rest, 2, visit)
}

// mapSchemaRefs calls visit for every reference to a UC schema from the resources
// that can have one, and stores the returned schema name.
func mapSchemaRefs(b *bundle.Bundle, visit schemaRefVisitor) {
	r := &b.Config.Resources

	for _, v := range r.Volumes {
		if v != nil {
			v.SchemaName = visit(&v.Lifecycle, v.CatalogName, v.SchemaName)
		}
	}
	for _, s := range r.Secrets {
		if s != nil {
			s.SchemaName = visit(&s.Lifecycle, s.CatalogName, s.SchemaName)
		}
	}
	for _, rm := range r.RegisteredModels {
		if rm != nil {
			rm.SchemaName = visit(&rm.Lifecycle, rm.CatalogName, rm.SchemaName)
		}
	}
	for _, p := range r.Pipelines {
		if p != nil {
			// "schema" and "target" have the same semantics in the SDP API but are mutually
			// exclusive i.e. only one can be set at a time.
			p.Schema = visit(&p.Lifecycle, p.Catalog, p.Schema)
			p.Target = visit(&p.Lifecycle, p.Catalog, p.Target)
		}
	}
	for _, qm := range r.QualityMonitors {
		if qm != nil {
			// OutputSchemaName is a "catalog.schema" name.
			qm.OutputSchemaName = visitCompoundSchemaRef(&qm.Lifecycle, qm.OutputSchemaName, 2, visit)
		}
	}
	for _, idx := range r.VectorSearchIndexes {
		if idx != nil {
			// Name is a "catalog.schema.index" name.
			idx.Name = visitCompoundSchemaRef(&idx.Lifecycle, idx.Name, 3, visit)
		}
	}
	for _, f := range r.Features {
		if f != nil {
			// FullName is a "catalog.schema.feature" name.
			f.FullName = visitCompoundSchemaRef(&f.Lifecycle, f.FullName, 3, visit)
		}
	}
	for _, mse := range r.ModelServingEndpoints {
		if mse == nil {
			continue
		}
		if mse.AiGateway != nil && mse.AiGateway.InferenceTableConfig != nil {
			itc := mse.AiGateway.InferenceTableConfig
			itc.SchemaName = visit(&mse.Lifecycle, itc.CatalogName, itc.SchemaName)
		}
		// AutoCaptureConfig is deprecated but still in use.
		if mse.Config != nil && mse.Config.AutoCaptureConfig != nil {
			acc := mse.Config.AutoCaptureConfig
			acc.SchemaName = visit(&mse.Lifecycle, acc.CatalogName, acc.SchemaName)
		}
	}
	for _, ms := range r.ModelServices {
		if ms != nil {
			ms.Parent = visitAiGatewayParent(&ms.Lifecycle, ms.Parent, visit)
		}
	}
	for _, ms := range r.McpServices {
		if ms != nil {
			ms.Parent = visitAiGatewayParent(&ms.Lifecycle, ms.Parent, visit)
		}
	}
	for _, mps := range r.ModelProviderServices {
		if mps != nil {
			mps.Parent = visitAiGatewayParent(&mps.Lifecycle, mps.Parent, visit)
		}
	}
}

func addDependsOn(lc *resources.Lifecycle, node string) {
	d := resources.DependsOn{Resource: node}
	if !slices.Contains(lc.DependsOn, d) {
		lc.DependsOn = append(lc.DependsOn, d)
	}
}

// dependsOnUC adds lifecycle.depends_on entries for the bundle schema and catalog that match
// the given literal names. It returns schemaName unchanged.
func dependsOnUC(b *bundle.Bundle, lc *resources.Lifecycle, catalogName, schemaName string) string {
	if k, s := findSchema(b, catalogName, schemaName); s != nil {
		addDependsOn(lc, "schemas."+k)
	}
	if k, c := findCatalog(b, catalogName); c != nil {
		addDependsOn(lc, "catalogs."+k)
	}
	return schemaName
}

func (m *captureUCDependencies) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	mapSchemaRefs(b, func(lc *resources.Lifecycle, catalogName, schemaName string) string {
		return dependsOnUC(b, lc, catalogName, schemaName)
	})

	for _, s := range b.Config.Resources.Schemas {
		if s != nil {
			dependsOnUC(b, &s.Lifecycle, s.CatalogName, "")
		}
	}
	return nil
}
