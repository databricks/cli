package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type captureSchemaDependency struct{}

// If a user defines a UC schema in the bundle, they can refer to it in DLT pipelines
// or UC Volumes using the `${resources.schemas.<schema_key>.name}` syntax. Using this
// syntax allows TF to capture the deploy time dependency this DLT pipeline or UC Volume
// has on the schema and deploy changes to the schema before deploying the pipeline or volume.
//
// This mutator translates any implicit schema references in DLT pipelines or UC Volumes
// to the explicit syntax.
func CaptureSchemaDependency() bundle.Mutator {
	return &captureSchemaDependency{}
}

func (m *captureSchemaDependency) Name() string {
	return "CaptureSchemaDependency"
}

func schemaNameRef(key string) string {
	return fmt.Sprintf("${resources.schemas.%s.name}", key)
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
	schemaK, schema := findSchema(b, v.CatalogName, v.SchemaName)
	if schema == nil {
		return
	}

	v.SchemaName = schemaNameRef(schemaK)
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

func (m *captureSchemaDependency) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, p := range b.Config.Resources.Pipelines {
		// "schema" and "target" have the same semantics in the DLT API but are mutually
		// exclusive i.e. only one can be set at a time. If schema is set, the pipeline
		// is in direct publishing mode and can write tables to multiple schemas
		// (vs target which is limited to a single schema).
		resolvePipelineTarget(p, b)
		resolvePipelineSchema(p, b)
	}
	for _, v := range b.Config.Resources.Volumes {
		resolveVolume(v, b)
	}
	return nil
}
