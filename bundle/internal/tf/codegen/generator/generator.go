// Package generator produces Go types from the Terraform provider schema.
package generator

import (
	"context"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	schemapkg "github.com/databricks/cli/bundle/internal/tf/codegen/schema"
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

	defer func() { _ = f.Close() }()

	return tmpl.Execute(f, c)
}

type root struct {
	OutputFile                 string
	ProviderVersion            string
	ProviderChecksumLinuxAmd64 string
	ProviderChecksumLinuxArm64 string
}

func (r *root) Generate(path string) error {
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf("./templates/%s.tmpl", r.OutputFile)))
	f, err := os.Create(filepath.Join(path, r.OutputFile))
	if err != nil {
		return err
	}

	defer func() { _ = f.Close() }()

	return tmpl.Execute(f, r)
}

// Run generates Go type files under path for every resource and data source in schema.
func Run(_ context.Context, schema *tfjson.ProviderSchema, checksums *schemapkg.ProviderChecksums, path string) error {
	// Generate types for resources
	var resources []*namedBlock
	for _, k := range slices.Sorted(maps.Keys(schema.ResourceSchemas)) {
		// Skipping all plugin framework struct generation.
		// TODO: This is a temporary fix, generation should be fixed in the future.
		if strings.HasSuffix(k, "_pluginframework") {
			continue
		}

		v := schema.ResourceSchemas[k]
		b := &namedBlock{
			filePattern:    "resource_%s.go",
			typeNamePrefix: "Resource",
			name:           k,
			block:          v.Block,
		}

		// Skip fields generation for resource_quality_monitor to avoid unwanted changes,
		// as of August 2025 the generator turns pointer fields into slices, which breaks the resource behaviour.
		// See https://github.com/databricks/cli/pull/3462
		if k == "databricks_quality_monitor" {
			log.Printf("Warning: Skipping file generation for %s to avoid known unwanted changes", k)
		} else {
			err := b.Generate(path)
			if err != nil {
				return err
			}
		}
		resources = append(resources, b)
	}

	// Generate types for data sources.
	var dataSources []*namedBlock
	for _, k := range slices.Sorted(maps.Keys(schema.DataSourceSchemas)) {
		// Skipping all plugin framework struct generation.
		// TODO: This is a temporary fix, generation should be fixed in the future.
		if strings.HasSuffix(k, "_pluginframework") {
			continue
		}

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

	// Generate root.go
	{
		r := &root{
			OutputFile:                 "root.go",
			ProviderVersion:            schemapkg.ProviderVersion,
			ProviderChecksumLinuxAmd64: checksums.LinuxAmd64,
			ProviderChecksumLinuxArm64: checksums.LinuxArm64,
		}
		err := r.Generate(path)
		if err != nil {
			return err
		}
	}

	return nil
}
