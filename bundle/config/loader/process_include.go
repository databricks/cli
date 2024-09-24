package loader

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// Steps:
// 1. Return info diag here if convention not followed
// 2. Add unit test for this mutator that convention is followed. Also add mutators for the dynamic extensions computation.
// 3. Add INFO rendering to the validate command
// 4. Add unit test that the INFO rendering is correct
// 5. Manually test the info diag.

// TODO: Should we detect and enforce this convention for .yaml files as well?

// TODO: Since we are skipping environemnts here, we should return a warning
// if environemnts is used (is that already the case?). And explain in the PR that
// we are choosing to not gather resources from environments.

// TODO: Talk in the PR about how this synergizes with the validate all unique
// keys mutator.
// Should I add a new abstraction for dyn values here?

var resourceTypes = []string{
	"job",
	"pipeline",
	"model",
	"experiment",
	"model_serving_endpoint",
	"registered_model",
	"quality_monitor",
	"schema",
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

	valid := true
	seenKeys := map[string]struct{}{}
	for _, rr := range resources {
		if len(seenKeys) == 0 {
			seenKeys[rr.key] = struct{}{}
			continue
		}

		if _, ok := seenKeys[rr.key]; !ok {
			valid = false
			break
		}

		if rr.typ != typ {
			valid = false
			break
		}
	}

	// The file only contains one resource defined in its resources or targets block,
	// and the resource is of the correct type.
	if valid {
		return nil
	}

	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("We recommend only defining a single %s when a file has the %s extension.", typ, ext))
	msg.WriteString("The following resources are defined or configured in this file:\n")
	for _, r := range resources {
		msg.WriteString(fmt.Sprintf("  - %s (%s)\n", r.key, r.typ))
	}

	locations := []dyn.Location{}
	paths := []dyn.Path{}
	for _, rr := range resources {
		locations = append(locations, rr.value.Locations()...)
		paths = append(paths, rr.path)
	}

	return diag.Diagnostics{
		{
			Severity:  diag.Info,
			Summary:   msg.String(),
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

	err := b.Config.Merge(this)
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}
