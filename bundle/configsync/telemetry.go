package configsync

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// recoverTelemetry swallows any panic raised while collecting or emitting
// telemetry. Telemetry is best-effort and must never fail the command, so every
// telemetry entry point that runs in the command path defers this.
func recoverTelemetry(ctx context.Context) {
	if r := recover(); r != nil {
		log.Debugf(ctx, "config-remote-sync telemetry panicked and was skipped: %v", r)
	}
}

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

	RecreateForcingChanges int64
	OverwrittenLocalEdits  int64

	FilesChangedCount int64
	FilesWrittenCount int64

	Restore RestoreStats

	ErrorMessage  string
	ErrorCategory protos.BundleConfigRemoteSyncErrorCategory
}

// RestoreStats counts the variable-reference restorations that can leak a
// current-target-scoped reference into a shared file. The safe paths (kept /
// compound-realigned) are not counted.
type RestoreStats struct {
	Retargeted   int64
	FromSiblings int64
}

// incRetargeted and incFromSiblings are nil-safe so that threading the pointer
// through the deep restoration recursion (or passing nil when counters are not
// needed) can never panic the command.
func (s *RestoreStats) incRetargeted() {
	if s != nil {
		s.Retargeted++
	}
}

func (s *RestoreStats) incFromSiblings() {
	if s != nil {
		s.FromSiblings++
	}
}

// CollectChangeStats fills change counters from the raw (pre-restoration)
// detected changes.
func (s *Stats) CollectChangeStats(ctx context.Context, changes Changes) {
	defer recoverTelemetry(ctx)
	if s.PerResourceType == nil {
		s.PerResourceType = make(map[string]*protos.BundleConfigRemoteSyncResourceChanges)
	}
	for resourceKey, resourceChanges := range changes {
		resourceType := resourceTypeFromKey(resourceKey)
		perType := s.PerResourceType[resourceType]
		if perType == nil {
			perType = &protos.BundleConfigRemoteSyncResourceChanges{ResourceType: resourceType}
			s.PerResourceType[resourceType] = perType
		}
		// Only Add/Replace/Remove reach this function: ChangesFromPlan filters
		// out Skip operations and convertChangeDesc never produces Unknown,
		// so the totals always equal the per-operation breakdown.
		for fieldPath, change := range resourceChanges {
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
			if isRecreateForcing(resourceType, fieldPath) {
				s.RecreateForcingChanges++
			}
			if change.LocalEdit {
				s.OverwrittenLocalEdits++
			}
		}
	}
}

// isRecreateForcing reports whether a change to fieldPath on resourceType lands
// on an immutable field (recreate_on_changes), meaning the next deploy will
// delete+recreate the resource. It consults both the hand-written and the
// spec-generated lifecycle configs, matching bundle plan behavior.
func isRecreateForcing(resourceType, fieldPath string) bool {
	node, err := structpath.ParsePath(fieldPath)
	if err != nil {
		return false
	}
	for _, cfg := range []*dresources.Config{dresources.MustLoadConfig(), dresources.MustLoadGeneratedConfig()} {
		rc, ok := cfg.Resources[resourceType]
		if !ok {
			continue
		}
		for _, rule := range rc.RecreateOnChanges {
			if node.HasPatternPrefix(rule.Field) {
				return true
			}
		}
	}
	return false
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

// LogTelemetry emits the BundleConfigRemoteSyncEvent for this run.
func (s *Stats) LogTelemetry(ctx context.Context) {
	defer recoverTelemetry(ctx)
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
			RecreateForcingChanges: s.RecreateForcingChanges,
			OverwrittenLocalEdits:  s.OverwrittenLocalEdits,
			ResourceChanges:        resourceChanges,
			FilesChangedCount:      s.FilesChangedCount,
			FilesWrittenCount:      s.FilesWrittenCount,
			RefsRetargeted:         s.Restore.Retargeted,
			RefsFromSiblings:       s.Restore.FromSiblings,
			ErrorMessage:           s.ErrorMessage,
			ErrorCategory:          s.ErrorCategory,
		},
	})
}
