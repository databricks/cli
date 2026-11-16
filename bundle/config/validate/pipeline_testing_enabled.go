package validate

import (
	"context"
	"fmt"

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
	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("testing"))
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// "testing:" without a value is not set, as in the typed configuration.
		if v.Kind() == dyn.KindNil {
			return v, nil
		}
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("the testing property of pipelines is experimental. Set %s=true to enable it", env.ExperimentalPipelineTestingVariable),
			Locations: v.Locations(),
			Paths:     dyn.ToStructPaths(p),
		})
		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}
