package loader

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"golang.org/x/exp/maps"
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

type resource struct {
	typ string
	l   dyn.Location
	p   dyn.Path
}

func gatherResources(r *config.Root) (map[string]resource, error) {
	res := map[string]resource{}

	// Gather all resources defined in the "resources" block.
	_, err := dyn.MapByPattern(
		r.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// The key for the resource. Eg: "my_job" for jobs.my_job.
			k := p[2].Key()
			// The type of the resource. Eg: "job" for jobs.my_job.
			typ := strings.TrimSuffix(p[1].Key(), "s")

			// We don't care if duplicate keys are defined across resources. That'll
			// cause an error that is caught by a separate mutator that validates
			// unique keys across all resources.
			res[k] = resource{typ: typ, l: v.Location(), p: p}
			return v, nil
		})
	if err != nil {
		return nil, err
	}

	// Gather all resources defined in a target block.
	_, err = dyn.MapByPattern(
		r.Value(),
		dyn.NewPattern(dyn.Key("targets"), dyn.AnyKey(), dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			k := p[4].Key()
			typ := strings.TrimSuffix(p[3].Key(), "s")

			res[k] = resource{typ: typ, l: v.Location(), p: p}
			return v, nil
		},
	)
	return res, err
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

	for _, typ := range resourceTypes {
		ext := fmt.Sprintf("%s.yml", typ)

		// File does not match this resource type. Check the next one.
		if !strings.HasSuffix(m.relPath, ".yml") {
			continue
		}

		resources, err := gatherResources(this)
		if err != nil {
			return diag.FromErr(err)
		}

		// file only has one resource defined, and the resource is of the correct
		// type. Thus it matches the recommendation we have for extensions like
		// .job.yml, .pipeline.yml, etc.
		keys := maps.Keys(resources)
		if len(resources) == 1 && resources[keys[0]].typ == typ {
			continue
		}

		msg := strings.Builder{}
		msg.WriteString(fmt.Sprintf("We recommend only defining a single %s in a file with the extension %s.\n", typ, ext))
		msg.WriteString("The following resources are defined or configured in this file:\n")
		for k, v := range resources {
			msg.WriteString(fmt.Sprintf("  - %s (%s)\n", k, v.typ))
		}

		locations := []dyn.Location{}
		paths := []dyn.Path{}
		for _, r := range resources {
			locations = append(locations, []dyn.Location{r.l}...)
			paths = append(paths, []dyn.Path{r.p}...)
		}

		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Info,
			Summary:   msg.String(),
			Locations: locations,
			Paths:     paths,
		})
	}

	err := b.Config.Merge(this)
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}
	return diags
}
