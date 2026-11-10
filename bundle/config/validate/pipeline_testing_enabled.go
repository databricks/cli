package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type pipelineTestingEnabled struct{ bundle.RO }

// PipelineTestingEnabled rejects the testing property of pipelines while the experimental feature is off.
func PipelineTestingEnabled() bundle.ReadOnlyMutator {
	return &pipelineTestingEnabled{}
}

func (v *pipelineTestingEnabled) Name() string {
	return "validate:pipeline_testing_enabled"
}

func (v *pipelineTestingEnabled) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if env.ExperimentalPipelineTesting(ctx) {
		return nil
	}
	var diags diag.Diagnostics
	pipelines := b.Config.Resources.Pipelines
	for _, key := range slices.Sorted(maps.Keys(pipelines)) {
		if pipelines[key] == nil || pipelines[key].Testing == nil {
			continue
		}
		path := dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(key), dyn.Key("testing"))
		value, err := dyn.GetByPath(b.Config.Value(), path)
		if err != nil {
			return diag.FromErr(err)
		}
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("the testing property of pipelines is experimental. Set %s=true to enable it", env.ExperimentalPipelineTestingVariable),
			Locations: value.Locations(),
			Paths:     dyn.ToStructPaths(path),
		})
	}
	return diags
}
