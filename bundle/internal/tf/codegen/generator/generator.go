package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tfjson "github.com/hashicorp/terraform-json"
)

func normalizeName(name string) string {
	return strings.TrimPrefix(name, "databricks_")
}

type collection struct {
	OutputFile string
	Blocks     []*namedBlock
}

func (c *collection) Generate(path string) error {
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf("./templates/%s.tmpl", c.OutputFile)))
	f, err := os.Create(filepath.Join(path, c.OutputFile))
	if err != nil {
		return err
	}

	defer f.Close()

	return tmpl.Execute(f, c)
}

func Run(ctx context.Context, schema *tfjson.ProviderSchema, path string) error {
	// Generate types for resources.
	var resources []*namedBlock
	for _, k := range sortKeys(schema.ResourceSchemas) {
		v := schema.ResourceSchemas[k]
		b := &namedBlock{
			filePattern:    "resource_%s.go",
			typeNamePrefix: "Resource",
			name:           k,
			block:          v.Block,
		}
		err := b.Generate(path)
		if err != nil {
			return err
		}
		resources = append(resources, b)
	}

	// Generate types for data sources.
	var dataSources []*namedBlock
	for _, k := range sortKeys(schema.DataSourceSchemas) {
		v := schema.DataSourceSchemas[k]
		b := &namedBlock{
			filePattern:    "data_source_%s.go",
			typeNamePrefix: "DataSource",
			name:           k,
			block:          v.Block,
		}
		err := b.Generate(path)
		if err != nil {
			return err
		}
		dataSources = append(dataSources, b)
	}

	// Generate type for provider configuration.
	{
		b := &namedBlock{
			filePattern:    "%s.go",
			typeNamePrefix: "",
			name:           "config",
			block:          schema.ConfigSchema.Block,
		}
		err := b.Generate(path)
		if err != nil {
			return err
		}
	}

	// Generate resources.go
	{
		cr := &collection{
			OutputFile: "resources.go",
			Blocks:     resources,
		}
		err := cr.Generate(path)
		if err != nil {
			return err
		}
	}

	// Generate data_sources.go
	{
		cr := &collection{
			OutputFile: "data_sources.go",
			Blocks:     dataSources,
		}
		err := cr.Generate(path)
		if err != nil {
			return err
		}
	}

	return nil
}
