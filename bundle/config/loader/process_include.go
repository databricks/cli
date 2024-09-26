package loader

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

var resourceTypes = []string{
	"job",
	"pipeline",
	"model",
	"experiment",
	"model_serving_endpoint",
	"registered_model",
	"quality_monitor",
	"schema",
	"cluster",
}

func validateFileFormat(r *config.Root, filePath string) diag.Diagnostics {
	for _, typ := range resourceTypes {
		for _, ext := range []string{fmt.Sprintf(".%s.yml", typ), fmt.Sprintf(".%s.yaml", typ)} {
			if strings.HasSuffix(filePath, ext) {
				return validateSingleResourceDefined(r, ext, typ)
			}
		}
	}

	return nil
}

func validateSingleResourceDefined(r *config.Root, ext, typ string) diag.Diagnostics {
	type resource struct {
		path  dyn.Path
		value dyn.Value
		typ   string
		key   string
	}

	resources := []resource{}

	// Gather all resources defined in the resources block.
	_, err := dyn.MapByPattern(
		r.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// The key for the resource. Eg: "my_job" for jobs.my_job.
			k := p[2].Key()
			// The type of the resource. Eg: "job" for jobs.my_job.
			typ := strings.TrimSuffix(p[1].Key(), "s")

			resources = append(resources, resource{path: p, value: v, typ: typ, key: k})
			return v, nil
		})
	if err != nil {
		return diag.FromErr(err)
	}

	// Gather all resources defined in a target block.
	_, err = dyn.MapByPattern(
		r.Value(),
		dyn.NewPattern(dyn.Key("targets"), dyn.AnyKey(), dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// The key for the resource. Eg: "my_job" for jobs.my_job.
			k := p[4].Key()
			// The type of the resource. Eg: "job" for jobs.my_job.
			typ := strings.TrimSuffix(p[3].Key(), "s")

			resources = append(resources, resource{path: p, value: v, typ: typ, key: k})
			return v, nil
		})
	if err != nil {
		return diag.FromErr(err)
	}

	typeMatch := true
	seenKeys := map[string]struct{}{}
	for _, rr := range resources {
		// case: The resource is not of the correct type.
		if rr.typ != typ {
			typeMatch = false
			break
		}

		seenKeys[rr.key] = struct{}{}
	}

	// Format matches. There's at most one resource defined in the file.
	// The resource is also of the correct type.
	if typeMatch && len(seenKeys) <= 1 {
		return nil
	}

	detail := strings.Builder{}
	detail.WriteString("The following resources are defined or configured in this file:\n")
	lines := []string{}
	for _, r := range resources {
		lines = append(lines, fmt.Sprintf("  - %s (%s)\n", r.key, r.typ))
	}
	// Sort the line s to print to make the output deterministic.
	sort.Strings(lines)
	// Compact the lines before writing them to the message to remove any duplicate lines.
	// This is needed because we do not dedup earlier when gathering the resources
	// and it's valid to define the same resource in both the resources and targets block.
	lines = slices.Compact(lines)
	for _, l := range lines {
		detail.WriteString(l)
	}

	locations := []dyn.Location{}
	paths := []dyn.Path{}
	for _, rr := range resources {
		locations = append(locations, rr.value.Locations()...)
		paths = append(paths, rr.path)
	}
	// Sort the locations and paths to make the output deterministic.
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].String() < locations[j].String()
	})
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].String() < paths[j].String()
	})

	return diag.Diagnostics{
		{
			Severity:  diag.Info,
			Summary:   fmt.Sprintf("We recommend only defining a single %s in a file with the %s extension.", typ, ext),
			Detail:    detail.String(),
			Locations: locations,
			Paths:     paths,
		},
	}
}

type processInclude struct {
	fullPath string
	relPath  string
}

// ProcessInclude loads the configuration at [fullPath] and merges it into the configuration.
func ProcessInclude(fullPath, relPath string) bundle.Mutator {
	return &processInclude{
		fullPath: fullPath,
		relPath:  relPath,
	}
}

func (m *processInclude) Name() string {
	return fmt.Sprintf("ProcessInclude(%s)", m.relPath)
}

func (m *processInclude) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	this, diags := config.Load(m.fullPath)
	if diags.HasError() {
		return diags
	}

	// Add any diagnostics associated with the file format.
	diags = append(diags, validateFileFormat(this, m.relPath)...)
	if diags.HasError() {
		return diags
	}

	err := b.Config.Merge(this)
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}
