package loader

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

func validateFileFormat(configRoot *config.Root, filePath string) diag.Diagnostics {
	for _, resourceDescription := range config.SupportedResources() {
		singularName := resourceDescription.SingularName

		for _, yamlExt := range []string{"yml", "yaml"} {
			ext := fmt.Sprintf(".%s.%s", singularName, yamlExt)
			if strings.HasSuffix(filePath, ext) {
				return validateSingleResourceDefined(configRoot, ext, singularName)
			}
		}
	}

	return nil
}

func validateSingleResourceDefined(configRoot *config.Root, ext, typ string) diag.Diagnostics {
	type resource struct {
		path *structpath.PathNode
		typ  string
		key  string
	}

	var resources []resource
	supportedResources := config.SupportedResources()

	// Gather all resources defined in the resources block, and in target blocks.
	configRoot.Resources.IterateAllResources(func(group, name string, _ config.ConfigResource) bool {
		resources = append(resources, resource{
			path: structpath.NewPath(nil, "resources", group, name),
			typ:  supportedResources[group].SingularName,
			key:  name,
		})
		return true
	})
	for _, target := range slices.Sorted(maps.Keys(configRoot.Targets)) {
		t := configRoot.Targets[target]
		if t == nil || t.Resources == nil {
			continue
		}
		t.Resources.IterateAllResources(func(group, name string, _ config.ConfigResource) bool {
			resources = append(resources, resource{
				path: structpath.NewPath(nil, "targets", target, "resources", group, name),
				typ:  supportedResources[group].SingularName,
				key:  name,
			})
			return true
		})
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

	var locations []diag.Location
	var paths []*structpath.PathNode
	for _, rr := range resources {
		locations = append(locations, configRoot.GetLocationsOfExact(rr.path)...)
		paths = append(paths, rr.path)
	}
	// Sort the locations and paths to make the output deterministic.
	slices.SortFunc(locations, func(a, b diag.Location) int {
		return cmp.Compare(a.String(), b.String())
	})
	slices.SortFunc(paths, func(a, b *structpath.PathNode) int {
		return cmp.Compare(a.String(), b.String())
	})

	return diag.Diagnostics{
		{
			Severity:  diag.Recommendation,
			Summary:   fmt.Sprintf("define a single %s in a file with the %s extension.", strings.ReplaceAll(typ, "_", " "), ext),
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
	this, diags := m.load()
	if diags.HasError() {
		return diags
	}

	err := b.Config.Merge(this)
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}

// load loads and validates the included file.
func (m *processInclude) load() (*config.Root, diag.Diagnostics) {
	this, diags := config.Load(m.fullPath)
	if diags.HasError() {
		return nil, diags
	}

	// Add any diagnostics associated with the file format.
	diags = append(diags, validateFileFormat(this, m.relPath)...)
	if diags.HasError() {
		return nil, diags
	}

	if len(this.Include) > 0 {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Include section is defined outside root file",
			Detail: `An include section is defined in a file that is not databricks.yml.
Only includes defined in databricks.yml are applied.`,
			Locations: this.GetLocations("include"),
			Paths:     structpath.NewPathSlice("include"),
		})
	}

	return this, diags
}
