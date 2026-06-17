package loader

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
)

func validateFileFormat(ctx context.Context, configRoot dyn.Value, filePath string) error {
	for _, resourceDescription := range config.SupportedResources() {
		singularName := resourceDescription.SingularName

		for _, yamlExt := range []string{"yml", "yaml"} {
			ext := fmt.Sprintf(".%s.%s", singularName, yamlExt)
			if strings.HasSuffix(filePath, ext) {
				return validateSingleResourceDefined(ctx, configRoot, ext, singularName)
			}
		}
	}

	return nil
}

func validateSingleResourceDefined(ctx context.Context, configRoot dyn.Value, ext, typ string) error {
	type resource struct {
		path  dyn.Path
		value dyn.Value
		typ   string
		key   string
	}

	var resources []resource
	supportedResources := config.SupportedResources()

	// Gather all resources defined in the resources block.
	_, err := dyn.MapByPattern(
		configRoot,
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// The key for the resource, e.g. "my_job" for jobs.my_job.
			k := p[2].Key()
			// The type of the resource, e.g. "job" for jobs.my_job.
			typ := supportedResources[p[1].Key()].SingularName

			resources = append(resources, resource{path: p, value: v, typ: typ, key: k})
			return v, nil
		})
	if err != nil {
		return err
	}

	// Gather all resources defined in a target block.
	_, err = dyn.MapByPattern(
		configRoot,
		dyn.NewPattern(dyn.Key("targets"), dyn.AnyKey(), dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// The key for the resource, e.g. "my_job" for jobs.my_job.
			k := p[4].Key()
			// The type of the resource, e.g. "job" for jobs.my_job.
			typ := supportedResources[p[3].Key()].SingularName

			resources = append(resources, resource{path: p, value: v, typ: typ, key: k})
			return v, nil
		})
	if err != nil {
		return err
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
	var lines []string
	for _, r := range resources {
		lines = append(lines, fmt.Sprintf("  - %s (%s)\n", r.key, r.typ))
	}
	// Sort the lines to print to make the output deterministic.
	slices.Sort(lines)
	// Compact the lines before writing them to the message to remove any duplicate lines.
	// This is needed because we do not dedup earlier when gathering the resources
	// and it's valid to define the same resource in both the resources and targets block.
	lines = slices.Compact(lines)
	for _, l := range lines {
		detail.WriteString(l)
	}

	var locations []dyn.Location
	var paths []dyn.Path
	for _, rr := range resources {
		locations = append(locations, rr.value.Locations()...)
		paths = append(paths, rr.path)
	}
	// Sort the locations and paths to make the output deterministic.
	slices.SortFunc(locations, func(a, b dyn.Location) int {
		return cmp.Compare(a.String(), b.String())
	})
	slices.SortFunc(paths, func(a, b dyn.Path) int {
		return cmp.Compare(a.String(), b.String())
	})

	logdiag.LogDiag(ctx, diag.Diagnostic{
		Severity:  diag.Recommendation,
		Summary:   fmt.Sprintf("define a single %s in a file with the %s extension.", strings.ReplaceAll(typ, "_", " "), ext),
		Detail:    detail.String(),
		Locations: locations,
		Paths:     paths,
	})
	return nil
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

func (m *processInclude) Apply(ctx context.Context, b *bundle.Bundle) error {
	this, diags := config.Load(m.fullPath)
	for _, d := range diags {
		if d.Severity != diag.Error {
			logdiag.LogDiag(ctx, d)
		}
	}
	if err := diags.Error(); err != nil {
		return err
	}

	// Add any diagnostics associated with the file format.
	if err := validateFileFormat(ctx, this.Value(), m.relPath); err != nil {
		return err
	}

	if len(this.Include) > 0 {
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Include section is defined outside root file",
			Detail: `An include section is defined in a file that is not databricks.yml.
Only includes defined in databricks.yml are applied.`,
			Locations: this.GetLocations("include"),
			Paths:     []dyn.Path{dyn.MustPathFromString("include")},
		})
	}

	return b.Config.Merge(this)
}
