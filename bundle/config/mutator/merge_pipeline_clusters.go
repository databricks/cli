package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/config/merge"
)

type mergePipelineClusters struct{}

func MergePipelineClusters() bundle.Mutator {
	return &mergePipelineClusters{}
}

func (m *mergePipelineClusters) Name() string {
	return "MergePipelineClusters"
}

func clusterLabel(cluster config.Value) (label string) {
	v := cluster.Get("label")
	if v == config.NilValue {
		return "default"
	}

	if v.Kind() != config.KindString {
		panic("cluster label must be a string")
	}

	return strings.ToLower(v.MustString())
}

func mergeClustersForPipeline(v config.Value) (config.Value, error) {
	clusters, ok := v.Get("clusters").AsSequence()
	if !ok {
		return v, nil
	}

	seen := make(map[string]config.Value)
	keys := make([]string, 0, len(clusters))

	// Target overrides are always appended, so we can iterate in natural order to
	// first find the base definition, and merge instances we encounter later.
	for i := range clusters {
		label := clusterLabel(clusters[i])

		// Register pipeline cluster with label if not yet seen before.
		ref, ok := seen[label]
		if !ok {
			keys = append(keys, label)
			seen[label] = clusters[i]
			continue
		}

		// Merge this instance into the reference.
		var err error
		seen[label], err = merge.Merge(ref, clusters[i])
		if err != nil {
			return v, err
		}
	}

	// Gather resulting clusters in natural order.
	out := make([]config.Value, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}

	return v.SetKey("clusters", config.NewValue(out, config.Location{})), nil
}

func (m *mergePipelineClusters) Apply(ctx context.Context, b *bundle.Bundle) error {

	// // MergeClusters merges cluster definitions with same label.
	// // The clusters field is a slice, and as such, overrides are appended to it.
	// // We can identify a cluster by its label, however, so we can use this label
	// // to figure out which definitions are actually overrides and merge them.
	// //
	// // Note: the cluster label is optional and defaults to 'default'.
	// // We therefore ALSO merge all clusters without a label.

	return b.Config.Mutate(func(v config.Value) (config.Value, error) {
		p := config.NewPathFromString("resources.pipelines")

		pv := v.Get("resources").Get("pipelines")
		pipelines, ok := pv.AsMap()
		if !ok {
			return v, nil
		}

		out := make(map[string]config.Value)
		for key, pipeline := range pipelines {
			var err error
			out[key], err = mergeClustersForPipeline(pipeline)
			if err != nil {
				return v, err
			}
		}

		v.Set(p, config.NewValue(out, config.Location{}))
	})
}
