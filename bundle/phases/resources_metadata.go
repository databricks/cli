package phases

import (
	"context"
	"os"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// collectResourcesMetadata builds a BundleResourcesMetadata for the deployment:
// per-resource-type counts, plus state sizes for the types the engine measures.
//
// Counts come from the configuration rather than from state, so every type is
// reported on every deploy whatever the engine, and a newly added resource type
// needs no telemetry change (AllResources is exhaustive by construction, guarded
// by TestResourcesAllResourcesCompleteness). That is what makes this message a
// superset of the deprecated per-type resource_*_count fields.
//
// Sizes come from b.Metrics.ResourceState, whose entries already carry
// StateSizeBytes from the WAL replay the direct deploy performed, so nothing is
// marshalled, read, or parsed here. Terraform leaves those zero.
//
// Returns nil when the bundle declares no resources and none are in state.
func collectResourcesMetadata(ctx context.Context, b *bundle.Bundle) *protos.BundleResourcesMetadata {
	resources := resourceMetadata(&b.Config.Resources, b.Metrics.ResourceState)
	if len(resources) == 0 {
		return nil
	}

	md := &protos.BundleResourcesMetadata{
		StateEngine: string(b.Metrics.StateEngine),
		Resources:   resources,
	}

	// Only the direct engine keeps the whole deployment state in one local file.
	// The terraform state file size is reported as terraform_state_size_bytes.
	if b.Metrics.StateEngine == engine.EngineDirect {
		md.StateFileSizeBytes = directStateFileSize(ctx, b)
	}

	return md
}

// resourceMetadata reports one entry per resource type, ordered by type name.
// Counts come from the configuration; max/mean/median state sizes are filled in
// for the types present in the deployment state.
func resourceMetadata(r *config.Resources, state resourcestate.ExportedResourcesMap) []protos.ResourceMetadata {
	counts := make(map[string]int64)
	for _, group := range r.AllResources() {
		if len(group.Resources) > 0 {
			counts[group.Description.PluralName] = int64(len(group.Resources))
		}
	}

	sizesByType := make(map[string][]int64)
	for key, rs := range state {
		t := config.GetResourceTypeFromKey(key)
		if t == "" {
			continue
		}
		sizesByType[t] = append(sizesByType[t], int64(rs.StateSizeBytes))
	}

	types := make([]string, 0, len(counts))
	for t := range counts {
		types = append(types, t)
	}
	for t, sizes := range sizesByType {
		if _, ok := counts[t]; !ok {
			// Sub-resources such as "jobs.permissions", and resources being removed
			// from the bundle: the configuration declares none, so the state is the
			// only available count.
			counts[t] = int64(len(sizes))
			types = append(types, t)
		}
	}
	slices.Sort(types)

	resources := make([]protos.ResourceMetadata, 0, len(types))
	for _, t := range types {
		m := protos.ResourceMetadata{
			ResourceType: t,
			Count:        counts[t],
		}
		if sizes := sizesByType[t]; len(sizes) > 0 {
			slices.Sort(sizes)
			m.StateSizeMaxBytes = statMax(sizes)
			m.StateSizeMeanBytes = statMean(sizes)
			m.StateSizeMedianBytes = statMedian(sizes)
		}
		resources = append(resources, m)
	}
	return resources
}

// statMax/statMean/statMedian operate on a slice already sorted ascending.
func statMax(sortedSizes []int64) int64 {
	return sortedSizes[len(sortedSizes)-1]
}

func statMean(sortedSizes []int64) int64 {
	var total int64
	for _, s := range sortedSizes {
		total += s
	}
	return total / int64(len(sortedSizes))
}

func statMedian(sortedSizes []int64) int64 {
	return sortedSizes[(len(sortedSizes)-1)/2]
}

// directStateFileSize returns the size in bytes of the direct engine's
// resources.json via a single stat (no read/parse), or 0 if it can't be stat'd.
func directStateFileSize(ctx context.Context, b *bundle.Bundle) int64 {
	_, localPath := b.StateFilenameDirect(ctx)
	info, err := os.Stat(localPath)
	if err != nil {
		log.Debugf(ctx, "resources-metadata telemetry: cannot stat direct state at %s: %s", localPath, err)
		return 0
	}
	return info.Size()
}
