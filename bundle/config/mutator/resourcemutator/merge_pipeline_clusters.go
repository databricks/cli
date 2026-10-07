package resourcemutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type mergePipelineClusters struct{}

func MergePipelineClusters() bundle.Mutator {
	return &mergePipelineClusters{}
}

func (m *mergePipelineClusters) Name() string {
	return "MergePipelineClusters"
}

func (m *mergePipelineClusters) clusterLabel(v structvar.View) string {
	switch v.Kind() {
	case structvar.KindInvalid, structvar.KindNil:
		// Note: the cluster label is optional and defaults to 'default'.
		// We therefore ALSO merge all clusters without a label.
		return "default"
	case structvar.KindString:
		s, _ := v.AsString()
		return strings.ToLower(s)
	default:
		panic("task key must be a string")
	}
}

func (m *mergePipelineClusters) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// The merge also lowercases labels and sets the "default" label, so only
	// lowercase labels are left unchanged.
	err := mergeByKey(b, "pipelines", b.Config.Resources.Pipelines, "clusters", "label",
		func(r *resources.Pipeline) []pipelines.PipelineCluster { return r.Clusters },
		func(c pipelines.PipelineCluster) string {
			if c.Label != strings.ToLower(c.Label) {
				return ""
			}
			return c.Label
		},
		m.clusterLabel, false)
	return diag.FromErr(err)
}
