package mutator

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

func findSchema(b *bundle.Bundle, catalogName, schemaName string) (string, *resources.Schema) {
	if catalogName == "" || schemaName == "" {
		return "", nil
	}

	for k, s := range b.Config.Resources.Schemas {
		if s.CreateSchema != nil && s.CatalogName == catalogName && s.Name == schemaName {
			return k, s
		}
	}
	return "", nil
}

func resolveVolume(v *resources.Volume, b *bundle.Bundle) {
	schemaK, schema := findSchema(b, v.CatalogName, v.SchemaName)
	if schema == nil {
		return
	}

	v.SchemaName = fmt.Sprintf("${resources.schemas.%s.name}", schemaK)
}

func resolvePipeline(p *resources.Pipeline, b *bundle.Bundle) {
	// schema and target have the same semantics in the DLT API but are mutually
	// exclusive. If schema is set, the pipeline is in direct publishing mode
	// and can write tables to multiple schemas (vs target which is limited to a single schema).
	schemaName := p.Schema
	if schemaName == "" {
		schemaName = p.Target
	}

	schemaK, schema := findSchema(b, p.Catalog, schemaName)
	if schema == nil {
		return
	}

	if p.Schema != "" {
		p.Schema = fmt.Sprintf("${resources.schemas.%s.name}", schemaK)
	} else if p.Target != "" {
		p.Target = fmt.Sprintf("${resources.schemas.%s.name}", schemaK)
	}
}

func (m *captureSchemaDependency) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, p := range b.Config.Resources.Pipelines {
		resolvePipeline(p, b)
	}
	for _, v := range b.Config.Resources.Volumes {
		resolveVolume(v, b)
	}
	return nil
}
