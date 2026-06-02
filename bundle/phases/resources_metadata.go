package phases

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// collectResourcesMetadata builds a BundleResourcesMetadata for the deploy.
//
// State sizes are computed by running each resource's typed config through
// the direct engine's adapter.PrepareState — the same transformation direct
// uses to derive the value it persists to resources.json — and marshaling
// each entry with dstate.SaveState's encoding (MarshalIndent("  ", " ")).
// The whole-file size is then computed by assembling those entries into a
// dstate.Database and marshaling it the way DeploymentState.unlockedSave
// writes it (MarshalIndent("", " ")). So:
//
//   - Under DATABRICKS_BUNDLE_ENGINE=direct, per-resource sizes equal
//     len(entry.State) on disk byte-for-byte, and state_file_size_bytes
//     matches the resources.json file size to within a few bytes (only
//     Lineage and Serial may differ, which we set to "" / 0 here).
//   - Under =terraform, the same computation runs against the bundle config,
//     producing identical numbers for the same logical bundle. tfstate is
//     never read.
//
// Returns nil when the bundle declares no resources.
func collectResourcesMetadata(ctx context.Context, b *bundle.Bundle) *protos.BundleResourcesMetadata {
	counts, sizesByType, fileSize := collectResourceCountsAndSizes(ctx, b)
	if len(counts) == 0 {
		return nil
	}

	types := unionKeys(counts, sizesByType)
	slices.Sort(types)

	resources := make([]protos.ResourceMetadata, 0, len(types))
	for _, t := range types {
		sizes := sizesByType[t]
		slices.SortFunc(sizes, func(a, b int64) int { return cmp.Compare(a, b) })
		resources = append(resources, protos.ResourceMetadata{
			ResourceType:         t,
			Count:                counts[t],
			StateSizeMaxBytes:    statMax(sizes),
			StateSizeMeanBytes:   statMean(sizes),
			StateSizeMedianBytes: statMedian(sizes),
		})
	}

	return &protos.BundleResourcesMetadata{
		StateEngine:        resolveDeployEngine(ctx, b),
		StateFileSizeBytes: fileSize,
		Resources:          resources,
	}
}

// collectResourceCountsAndSizes walks the bundle config and assembles a
// dstate.Database with each resource's PrepareState'd value, then marshals
// that database the way direct writes resources.json. Returns per-type
// counts, per-type per-resource byte lengths, and the byte length of the
// whole simulated state file.
func collectResourceCountsAndSizes(ctx context.Context, b *bundle.Bundle) (map[string]int64, map[string][]int64, int64) {
	counts := make(map[string]int64)
	sizesByType := make(map[string][]int64)

	adapters := getAdapters(ctx, b)
	db := dstate.NewDatabase("", 0)

	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if len(p) < 3 {
			return v, nil
		}
		resourceType := p[1].Key()
		counts[resourceType]++

		stateBytes, err := resourceStateBytes(b, adapters, p, resourceType)
		if err != nil {
			log.Debugf(ctx, "resources-metadata telemetry: %s: %s", p, err)
			return v, nil
		}
		sizesByType[resourceType] = append(sizesByType[resourceType], int64(len(stateBytes)))
		db.State[p.String()] = dstate.ResourceEntry{
			ID:    extractResourceID(v),
			State: stateBytes,
		}
		return v, nil
	})
	if err != nil {
		log.Debugf(ctx, "resources-metadata telemetry: failed to walk config resources: %s", err)
	}

	var fileSize int64
	if len(db.State) > 0 {
		raw, mErr := json.MarshalIndent(db, "", " ")
		if mErr != nil {
			log.Debugf(ctx, "resources-metadata telemetry: failed to marshal database envelope: %s", mErr)
		} else {
			fileSize = int64(len(raw))
		}
	}
	return counts, sizesByType, fileSize
}

// resourceStateBytes derives the bytes direct would store for one resource:
// GetResourceConfig (typed) → adapter.PrepareState → MarshalIndent with the
// same prefix/indent direct uses in dstate.SaveState. Falls back to marshaling
// the typed config when no adapter is registered for the resource type
// (e.g., a type the direct engine doesn't yet support).
func resourceStateBytes(b *bundle.Bundle, adapters map[string]*dresources.Adapter, p dyn.Path, resourceType string) ([]byte, error) {
	cfg, err := b.Config.GetResourceConfig(p.String())
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	target := cfg
	if adapter, ok := adapters[resourceType]; ok {
		state, err := adapter.PrepareState(cfg)
		if err != nil {
			return nil, fmt.Errorf("prepare state: %w", err)
		}
		target = state
	}

	// dstate.SaveState writes resource state with MarshalIndent using these
	// exact prefix/indent arguments; matching them here means each resource's
	// byte length equals len(entry.State) on disk for direct deploys.
	raw, err := json.MarshalIndent(target, "  ", " ")
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return raw, nil
}

// extractResourceID returns the resource's ID string from its dyn.Value entry,
// or "" if not yet set. Each resources.<type>.<name> entry has an "id" field
// populated post-deploy (via BaseResource.ID).
func extractResourceID(v dyn.Value) string {
	idVal, err := dyn.Get(v, "id")
	if err != nil || idVal.Kind() != dyn.KindString {
		return ""
	}
	return idVal.MustString()
}

// getAdapters returns adapters initialized for PrepareState. If the bundle
// already has them initialized (direct engine path), reuse them. Otherwise,
// build a fresh set with a nil workspace client — PrepareState is a pure
// transformation that doesn't touch the client.
func getAdapters(ctx context.Context, b *bundle.Bundle) map[string]*dresources.Adapter {
	if b.DeploymentBundle.Adapters != nil {
		return b.DeploymentBundle.Adapters
	}
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		log.Debugf(ctx, "resources-metadata telemetry: failed to init adapters: %s", err)
		return nil
	}
	return adapters
}

// resolveDeployEngine returns the effective deploy engine ("direct" or
// "terraform"). Mirrors cmd/bundle/utils.ResolveEngineSetting but is inlined
// here to avoid a layering import (bundle/phases must not depend on cmd/).
func resolveDeployEngine(ctx context.Context, b *bundle.Bundle) string {
	if b.Config.Bundle.Engine != engine.EngineNotSet {
		return string(b.Config.Bundle.Engine.ThisOrDefault())
	}
	envEngine, _ := engine.FromEnv(ctx)
	return string(envEngine.ThisOrDefault())
}

func unionKeys(a map[string]int64, b map[string][]int64) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func statMax(sortedSizes []int64) int64 {
	if len(sortedSizes) == 0 {
		return 0
	}
	return sortedSizes[len(sortedSizes)-1]
}

func statMean(sortedSizes []int64) int64 {
	if len(sortedSizes) == 0 {
		return 0
	}
	var total int64
	for _, s := range sortedSizes {
		total += s
	}
	return total / int64(len(sortedSizes))
}

func statMedian(sortedSizes []int64) int64 {
	if len(sortedSizes) == 0 {
		return 0
	}
	return sortedSizes[(len(sortedSizes)-1)/2]
}
