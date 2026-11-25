package resourcemutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type mergePipelineClusters struct{}

func MergePipelineClusters() bundle.Mutator {
	return &mergePipelineClusters{}
}

func (m *mergePipelineClusters) Name() string {
	return "MergePipelineClusters"
}

func (m *mergePipelineClusters) clusterLabel(v dyn.Value) string {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		// Note: the cluster label is optional and defaults to 'default'.
		// We therefore ALSO merge all clusters without a label.
		return "default"
	case dyn.KindString:
		return strings.ToLower(v.MustString())
	default:
		panic("task key must be a string")
	}
}

func (m *mergePipelineClusters) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Kind() == dyn.KindNil {
			return v, nil
		}

		return dyn.Map(v, "resources.pipelines", dyn.Foreach(func(_ dyn.Path, pipeline dyn.Value) (dyn.Value, error) {
			return dyn.Map(pipeline, "clusters", merge.ElementsByKey("label", m.clusterLabel))
		}))
	})

	return diag.FromErr(err)
}
