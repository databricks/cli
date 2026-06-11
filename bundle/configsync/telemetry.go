package configsync

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// ErrStateSnapshotNotFound indicates the deployed state snapshot required for
// change detection does not exist (the bundle was likely never deployed).
var ErrStateSnapshotNotFound = errors.New("state snapshot not found")

// Stats accumulates aggregate counters for a single config-remote-sync run.
// All values are counts, booleans, or enumerated categories; no resource
// names, paths, or configuration values are ever recorded.
type Stats struct {
	Save   bool
	Engine engine.EngineType

	ChangesTotal int64
	AddCount     int64
	ReplaceCount int64
	RemoveCount  int64

	// Keyed by resource type (e.g. "jobs"), as parsed from change keys like
	// "resources.jobs.foo".
	PerResourceType map[string]*protos.BundleConfigRemoteSyncResourceChanges

	FilesChangedCount int64
	FilesWrittenCount int64

	Restore RestoreStats

	RawValuesWithVarSyntax int64

	ErrorCategory protos.BundleConfigRemoteSyncErrorCategory
}

// RestoreStats counts variable-reference restorations by mechanism.
type RestoreStats struct {
	Kept         int64
	Compound     int64
	Retargeted   int64
	FromSiblings int64
}

// CollectChangeStats fills change counters from the raw (pre-restoration)
// detected changes.
func (s *Stats) CollectChangeStats(changes Changes) {
	if s.PerResourceType == nil {
		s.PerResourceType = make(map[string]*protos.BundleConfigRemoteSyncResourceChanges)
	}
	for resourceKey, resourceChanges := range changes {
		perType := s.PerResourceType[resourceTypeFromKey(resourceKey)]
		if perType == nil {
			perType = &protos.BundleConfigRemoteSyncResourceChanges{ResourceType: resourceTypeFromKey(resourceKey)}
			s.PerResourceType[perType.ResourceType] = perType
		}
		// Only Add/Replace/Remove reach this function: DetectChanges filters
		// out Skip operations and convertChangeDesc never produces Unknown,
		// so the totals always equal the per-operation breakdown.
		for _, change := range resourceChanges {
			s.ChangesTotal++
			perType.ChangesCount++
			switch change.Operation {
			case OperationAdd:
				s.AddCount++
				perType.AddCount++
			case OperationReplace:
				s.ReplaceCount++
				perType.ReplaceCount++
			case OperationRemove:
				s.RemoveCount++
				perType.RemoveCount++
			case OperationUnknown, OperationSkip:
			}
			s.RawValuesWithVarSyntax += countVarSyntax(change.Value)
		}
	}
}

// resourceTypeFromKey extracts the resource type from a change key like
// "resources.jobs.foo". Only the type segment is recorded; resource keys are
// never logged.
func resourceTypeFromKey(resourceKey string) string {
	parts := strings.SplitN(resourceKey, ".", 3)
	if len(parts) < 2 || parts[0] != "resources" {
		return "unknown"
	}
	return parts[1]
}

// countVarSyntax counts string leaves containing the literal "${" sequence.
// Such values are written to YAML verbatim and are subject to interpolation
// on the next deploy, so they measure exposure to escaping issues.
func countVarSyntax(value any) int64 {
	var n int64
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "${") {
			n++
		}
	case map[string]any:
		for _, val := range v {
			n += countVarSyntax(val)
		}
	case []any:
		for _, val := range v {
			n += countVarSyntax(val)
		}
	}
	return n
}

// LogTelemetry emits the BundleConfigRemoteSyncEvent for this run.
func (s *Stats) LogTelemetry(ctx context.Context) {
	resourceChanges := make([]protos.BundleConfigRemoteSyncResourceChanges, 0, len(s.PerResourceType))
	for _, perType := range s.PerResourceType {
		resourceChanges = append(resourceChanges, *perType)
	}
	slices.SortFunc(resourceChanges, func(a, b protos.BundleConfigRemoteSyncResourceChanges) int {
		return strings.Compare(a.ResourceType, b.ResourceType)
	})

	telemetry.Log(ctx, protos.DatabricksCliLog{
		BundleConfigRemoteSyncEvent: &protos.BundleConfigRemoteSyncEvent{
			Save:                   s.Save,
			Engine:                 string(s.Engine),
			ChangesTotal:           s.ChangesTotal,
			AddCount:               s.AddCount,
			ReplaceCount:           s.ReplaceCount,
			RemoveCount:            s.RemoveCount,
			ResourceChanges:        resourceChanges,
			FilesChangedCount:      s.FilesChangedCount,
			FilesWrittenCount:      s.FilesWrittenCount,
			RefsKept:               s.Restore.Kept,
			RefsCompound:           s.Restore.Compound,
			RefsRetargeted:         s.Restore.Retargeted,
			RefsFromSiblings:       s.Restore.FromSiblings,
			RawValuesWithVarSyntax: s.RawValuesWithVarSyntax,
			ErrorCategory:          s.ErrorCategory,
		},
	})
}
