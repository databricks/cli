package phases

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	tfjson "github.com/hashicorp/terraform-json"
)

// collectResourcesMetadata builds a BundleResourcesMetadata for the deploy:
// per-resource-type counts come from the bundle configuration (matching the
// semantics of the deprecated DatabricksBundleDeployEvent.resource_*_count
// fields), and state-size statistics come from the on-disk deployment state
// file. For Terraform deployments the tfstate is translated to the direct-
// engine representation before sizing so per-type stats are comparable across
// engines.
//
// Returns nil only on a complete absence of signal (no resources declared and
// no readable state). Telemetry must never fail a deploy — all parse errors
// are logged at debug level and treated as missing data.
//
// This file is the sole site of resource-metadata telemetry logic. To remove
// the feature: delete this file and its companion test, revert the call site
// in telemetry.go, and revert the ResourcesMetadata field in
// libs/telemetry/protos/bundle_deploy.go.
func collectResourcesMetadata(ctx context.Context, b *bundle.Bundle) *protos.BundleResourcesMetadata {
	counts := countResourcesByType(ctx, b)

	engine, fileSize, sizesByType := readStateForMetadata(ctx, b)

	if len(counts) == 0 && len(sizesByType) == 0 && fileSize == 0 {
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
		StateEngine:        engine,
		StateFileSizeBytes: fileSize,
		Resources:          resources,
	}
}

// countResourcesByType walks the bundle config and counts top-level resources
// at "resources.<type>.<name>". Returns map[type]count.
func countResourcesByType(ctx context.Context, b *bundle.Bundle) map[string]int64 {
	out := make(map[string]int64)
	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if len(p) >= 2 {
			out[p[1].Key()]++
		}
		return v, nil
	})
	if err != nil {
		log.Debugf(ctx, "resources-metadata telemetry: failed to walk config resources: %s", err)
	}
	return out
}

// readStateForMetadata reads whichever local state file exists (direct
// preferred, then terraform) and returns engine name, whole-file size, and
// per-resource-type sizes. Returns ("", 0, nil) if no state is present or if
// the bundle isn't far enough through initialization to have a target
// selected (which is required to compute state file paths).
func readStateForMetadata(ctx context.Context, b *bundle.Bundle) (string, int64, map[string][]int64) {
	if b.Target == nil {
		return "", 0, nil
	}

	if _, localPath := b.StateFilenameDirect(ctx); localPath != "" {
		raw, err := readStateFile(localPath)
		if err == nil && raw != nil {
			return "direct", int64(len(raw)), parseDirectStateSizes(ctx, raw)
		}
		if err != nil {
			log.Debugf(ctx, "resources-metadata telemetry: skipping direct state at %s: %s", localPath, err)
		}
	}

	if _, localPath := b.StateFilenameTerraform(ctx); localPath != "" {
		raw, err := readStateFile(localPath)
		if errors.Is(err, fs.ErrNotExist) {
			altPath := terraformCacheStatePath(ctx, b)
			if altPath != localPath && altPath != "" {
				raw, err = readStateFile(altPath)
			}
		}
		if err == nil && raw != nil {
			return "terraform", int64(len(raw)), parseTerraformStateSizes(ctx, raw)
		}
		if err != nil {
			log.Debugf(ctx, "resources-metadata telemetry: skipping terraform state at %s: %s", localPath, err)
		}
	}

	return "", 0, nil
}

func readStateFile(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return raw, err
}

func terraformCacheStatePath(ctx context.Context, b *bundle.Bundle) string {
	dir, err := terraform.Dir(ctx, b)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "terraform.tfstate")
}

func parseDirectStateSizes(ctx context.Context, raw []byte) map[string][]int64 {
	var db dstate.Database
	if err := json.Unmarshal(raw, &db); err != nil {
		log.Debugf(ctx, "resources-metadata telemetry: failed to parse direct state: %s", err)
		return nil
	}
	byType := make(map[string][]int64)
	for key, entry := range db.State {
		t := resourceTypeFromKey(key)
		if t == "" {
			continue
		}
		byType[t] = append(byType[t], int64(len(entry.State)))
	}
	return byType
}

func parseTerraformStateSizes(ctx context.Context, raw []byte) map[string][]int64 {
	var state struct {
		Version   int `json:"version"`
		Resources []struct {
			Type      string              `json:"type"`
			Mode      tfjson.ResourceMode `json:"mode"`
			Instances []struct {
				Attributes json.RawMessage `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		log.Debugf(ctx, "resources-metadata telemetry: failed to parse terraform state: %s", err)
		return nil
	}
	byType := make(map[string][]int64)
	for _, resource := range state.Resources {
		if resource.Mode != tfjson.ManagedResourceMode {
			continue
		}
		groupName, ok := terraform.TerraformToGroupName[resource.Type]
		if !ok {
			continue
		}
		for _, instance := range resource.Instances {
			byType[groupName] = append(byType[groupName], int64(len(instance.Attributes)))
		}
	}
	return byType
}

// resourceTypeFromKey extracts the resource type from a direct-engine state
// key. Direct-engine keys are of the form "resources.<type>.<name>" or
// "resources.<type>.<name>.<sub>" (for permissions/grants/secret_acls).
// Returns "" for keys that don't match.
func resourceTypeFromKey(key string) string {
	parts := strings.SplitN(key, ".", 4)
	if len(parts) < 3 || parts[0] != "resources" {
		return ""
	}
	if len(parts) == 4 {
		// Sub-resources like permissions / grants / secret_acls live at
		// "resources.<parent>.<name>.<sub>". Track them under the sub-resource
		// type so they aggregate across resource families.
		return parts[3]
	}
	return parts[1]
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
