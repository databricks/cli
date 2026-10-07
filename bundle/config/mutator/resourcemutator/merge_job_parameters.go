package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type mergeJobParameters struct{}

func MergeJobParameters() bundle.Mutator {
	return &mergeJobParameters{}
}

func (m *mergeJobParameters) Name() string {
	return "MergeJobParameters"
}

func (m *mergeJobParameters) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := mergeByKey(b, "jobs", b.Config.Resources.Jobs, "parameters", "name",
		func(r *resources.Job) []jobs.JobParameterDefinition { return r.Parameters },
		func(p jobs.JobParameterDefinition) string { return p.Name },
		stringKey("parameter name"), false)
	return diag.FromErr(err)
}
