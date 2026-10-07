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
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

func validateFileFormat(configRoot structvar.View, filePath string) diag.Diagnostics {
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

func validateSingleResourceDefined(configRoot structvar.View, ext, typ string) diag.Diagnostics {
	type resource struct {
		path  *structpath.PathNode
		value structvar.View
		typ   string
		key   string
	}

	var resources []resource
	supportedResources := config.SupportedResources()

	// Gather all resources defined in the resources block.
	err := structvar.ForEach(
		configRoot,
		structpath.MustParsePattern("resources.*.*"),
		func(np *structpath.PathNode, v structvar.View) error {
			// The key for the resource, e.g. "my_job" for jobs.my_job.
			k := np.KeyAt(2)
			// The type of the resource, e.g. "job" for jobs.my_job.
			typ := supportedResources[np.KeyAt(1)].SingularName

			resources = append(resources, resource{path: np, value: v, typ: typ, key: k})
			return nil
		})
	if err != nil {
		return diag.FromErr(err)
	}

	// Gather all resources defined in a target block.
	err = structvar.ForEach(
		configRoot,
		structpath.MustParsePattern("targets.*.resources.*.*"),
		func(np *structpath.PathNode, v structvar.View) error {
			// The key for the resource, e.g. "my_job" for jobs.my_job.
			k := np.KeyAt(4)
			// The type of the resource, e.g. "job" for jobs.my_job.
			typ := supportedResources[np.KeyAt(3)].SingularName

			resources = append(resources, resource{path: np, value: v, typ: typ, key: k})
			return nil
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
		locations = append(locations, rr.value.Locations()...)
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
	diags = append(diags, validateFileFormat(this.View(), m.relPath)...)
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
