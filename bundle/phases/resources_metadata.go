package phases

import (
	"context"
	"os"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// directEngine is the only engine for which resources_metadata is populated.
const directEngine = "direct"

// collectResourcesMetadata builds a BundleResourcesMetadata from the per-resource
// state sizes captured during the deploy.
//
// Only direct deploys are measured. b.Metrics.ResourceState is the direct
// engine's finalized state, populated in deployCore from the WAL replay the
// deploy already performs; each entry carries StateSizeBytes (len of the JSON
// blob stored in resources.json) and StateCompressedSizeBytes (its zstd-compressed
// length, computed during export). So no marshalling, file read, or JSON parsing
// happens here — sizes are read straight off the in-memory map. The whole-file
// size comes from a single os.Stat (no parse). Returns nil for terraform
// deploys (ResourceState is nil) and when no resources are in state.
func collectResourcesMetadata(ctx context.Context, b *bundle.Bundle) *protos.BundleResourcesMetadata {
	state := b.Metrics.ResourceState
	if len(state) == 0 {
		return nil
	}

	resources := resourceMetadataFromState(state)
	if len(resources) == 0 {
		return nil
	}

	return &protos.BundleResourcesMetadata{
		StateEngine:        directEngine,
		StateFileSizeBytes: directStateFileSize(ctx, b),
		Resources:          resources,
	}
}

// resourceMetadataFromState groups the deployment state by resource type and
// computes per-type count and max/mean/median state size, both raw and
// zstd-compressed. Sizes are sorted per type (needed for the median).
func resourceMetadataFromState(state resourcestate.ExportedResourcesMap) []protos.ResourceMetadata {
	sizesByType := make(map[string][]int64)
	compressedByType := make(map[string][]int64)
	for key, rs := range state {
		t := config.GetResourceTypeFromKey(key)
		if t == "" {
			continue
		}
		sizesByType[t] = append(sizesByType[t], int64(rs.StateSizeBytes))
		compressedByType[t] = append(compressedByType[t], int64(rs.StateCompressedSizeBytes))
	}

	types := make([]string, 0, len(sizesByType))
	for t := range sizesByType {
		types = append(types, t)
	}
	slices.Sort(types)

	resources := make([]protos.ResourceMetadata, 0, len(types))
	for _, t := range types {
		sizes := sizesByType[t]
		slices.Sort(sizes)
		compressed := compressedByType[t]
		slices.Sort(compressed)
		resources = append(resources, protos.ResourceMetadata{
			ResourceType:                   t,
			Count:                          int64(len(sizes)),
			StateSizeMaxBytes:              statMax(sizes),
			StateSizeMeanBytes:             statMean(sizes),
			StateSizeMedianBytes:           statMedian(sizes),
			StateCompressedSizeMaxBytes:    statMax(compressed),
			StateCompressedSizeMeanBytes:   statMean(compressed),
			StateCompressedSizeMedianBytes: statMedian(compressed),
		})
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
