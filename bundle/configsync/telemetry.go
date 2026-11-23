package configsync

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dresources"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

// recoverTelemetry swallows any panic raised while collecting or emitting
// telemetry. Telemetry is best-effort and must never fail the command, so every
// telemetry entry point that runs in the command path defers this.
func recoverTelemetry(ctx context.Context) {
	if r := recover(); r != nil {
		log.Debugf(ctx, "config-remote-sync telemetry panicked and was skipped: %v", r)
	}
}

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

	// Changes that were detected and reported but not written back, because they
	// could not be attributed to a single source location.
	SkippedChangesCount int64

	// Resources left unchanged because a cluster policy could not be read, by cause.
	ClusterPolicyNotFoundCount         int64
	ClusterPolicyPermissionDeniedCount int64
	ClusterPolicyReadFailedCount       int64

	Restore RestoreStats

	// Identity of the resource state the run actually read. Without these, a
	// "no deployed <type> resource with id" failure cannot be told apart from
	// reading a stale local cache, another target's state, or no state at all.
	StateSerial  int64
	StateLineage string
	StateSource  string
	// Pointer so the informative zero (no state file found at all) survives
	// omitempty on serialization.
	StatesAvailableCount *int64

	// Resource ids present in the state that was read, and the ids the run was
	// asked to sync. Comparing the two classifies a selector miss.
	StateResourceIDs    *protos.BundleConfigRemoteSyncResourceIds
	SelectedResourceIDs *protos.BundleConfigRemoteSyncResourceIds

	InGitRepo *bool

	ErrorMessage  string
	ErrorCategory protos.BundleConfigRemoteSyncErrorCategory
}

// CollectSkippedResources counts resources held back by cluster policy lookups, by
// reason. Only counts are kept; the resource keys and policy IDs stay in the output.
func (s *Stats) CollectSkippedResources(skipped []SkippedChange) {
	for _, skip := range skipped {
		switch skip.Reason {
		case SkipReasonClusterPolicyNotFound:
			s.ClusterPolicyNotFoundCount++
		case SkipReasonClusterPolicyPermissionDenied:
			s.ClusterPolicyPermissionDeniedCount++
		default:
			s.ClusterPolicyReadFailedCount++
		}
	}
}

// CollectStateStats records which resource state the run read. Only the
// state's identity is recorded (serial, system-generated lineage UUID,
// local/remote origin, candidate count) — never a path or a target name.
//
// None of these can contain PII: the lineage is an opaque random UUID minted by
// the state layer, and the source is one of two fixed literals.
// recordedVersion is the version the deployment metadata service holds, zero when the bundle does
// not record deployment history. A recorded state file persists no serial of its own, so the
// recorded version stands in for it and the field means the same thing for both backends.
func (s *Stats) CollectStateStats(desc *statemgmt.StateDesc, recordedVersion int) {
	if desc == nil {
		return
	}
	s.StateSerial = int64(desc.Serial)
	if recordedVersion > 0 {
		s.StateSerial = int64(recordedVersion)
	}
	s.StateLineage = desc.Lineage
	s.StateSource = "remote"
	if desc.IsLocal {
		s.StateSource = "local"
	}
	// AllStates is only populated when at least one state was found; a
	// synthesized empty state descriptor leaves it nil.
	n := int64(len(desc.AllStates))
	s.StatesAvailableCount = &n
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
		// Only Add/Replace/Remove reach this function: ExtractChanges filters
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

// resourceIDLimit caps how many ids of one type are reported. Mirrors
// phases.ResourceIdLimit: the telemetry upload has a short timeout, so a bundle
// with very many resources must not produce an unbounded payload.
const resourceIDLimit = 1000

// collectResourceIDs groups "<type>:<id>" pairs into the per-type id lists.
// Returns nil when nothing was collected so the field is omitted entirely.
func collectResourceIDs(typeAndIDs []string) *protos.BundleConfigRemoteSyncResourceIds {
	out := &protos.BundleConfigRemoteSyncResourceIds{}
	any := false
	for _, typeAndID := range typeAndIDs {
		resourceType, id, ok := strings.Cut(typeAndID, ":")
		if !ok || id == "" {
			continue
		}
		// Sub-resources (permissions/grants) are indexed under their parent's
		// type with a path-shaped object id; those are not selectable and the
		// path would be redacted, so skip them.
		if strings.Contains(id, "/") {
			continue
		}
		switch resourceType {
		case "jobs":
			if len(out.ResourceJobIDs) < resourceIDLimit {
				out.ResourceJobIDs = append(out.ResourceJobIDs, id)
				any = true
			}
		case "pipelines":
			if len(out.ResourcePipelineIDs) < resourceIDLimit {
				out.ResourcePipelineIDs = append(out.ResourcePipelineIDs, id)
				any = true
			}
		case "clusters":
			if len(out.ResourceClusterIDs) < resourceIDLimit {
				out.ResourceClusterIDs = append(out.ResourceClusterIDs, id)
				any = true
			}
		case "dashboards":
			if len(out.ResourceDashboardIDs) < resourceIDLimit {
				out.ResourceDashboardIDs = append(out.ResourceDashboardIDs, id)
				any = true
			}
		}
	}
	if !any {
		return nil
	}
	slices.Sort(out.ResourceJobIDs)
	slices.Sort(out.ResourcePipelineIDs)
	slices.Sort(out.ResourceClusterIDs)
	slices.Sort(out.ResourceDashboardIDs)
	return out
}

// CollectSelectedIDs records the ids the run was asked to sync.
func (s *Stats) CollectSelectedIDs(selectors []string) {
	s.SelectedResourceIDs = collectResourceIDs(selectors)
}

// CollectStateIDs records the ids present in the state that was read.
func (s *Stats) CollectStateIDs(typeAndIDs []string) {
	s.StateResourceIDs = collectResourceIDs(typeAndIDs)
}

// WithProfileAuth sets the auth config from an explicitly selected profile when
// loading the bundle or its target failed before the bundle set its own, so the
// failure can still be reported. Without the target its workspace is unknown, so
// nothing is set without a profile or when the profile is not found.
// Only the profile is used: the bundle config can name other hosts, and
// telemetry must not be sent there. The config is left for the upload to
// resolve within its own timeout, and as the upload may be cut off at exit, its
// auth never refreshes an OAuth token.
func WithProfileAuth(ctx context.Context, b *bundle.Bundle, profileName, target string) (out context.Context) {
	out = ctx
	defer recoverTelemetry(ctx)
	if cmdctx.HasConfigUsed(ctx) || telemetry.Disabled(ctx) {
		return ctx
	}
	// With the target selected, only the bundle's own auth can have failed, and
	// auth errors are not reported.
	if b != nil && b.Config.Bundle.Target != "" {
		log.Debugf(ctx, "config-remote-sync telemetry skipped: bundle auth failed")
		return ctx
	}
	profileName = cmp.Or(profileName, env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"))
	if profileName == "" {
		log.Debugf(ctx, "config-remote-sync telemetry skipped: no profile selected")
		return ctx
	}
	profiles, err := profile.GetProfiler(ctx).LoadProfiles(ctx, profile.WithName(profileName))
	if err != nil || len(profiles) == 0 {
		log.Debugf(ctx, "config-remote-sync telemetry skipped: profile %s not found", profileName)
		return ctx
	}
	// As in bundle auth, a profile for another host than the bundle's is rejected,
	// and so is one for another workspace on the same unified host.
	if b != nil {
		host, id := b.Config.Workspace.Host, b.Config.Workspace.WorkspaceID
		// The target was not merged, so its own workspace is not in the root config yet.
		if t := b.Config.Targets[selectedTarget(ctx, b, target)]; t != nil && t.Workspace != nil {
			host, id = cmp.Or(t.Workspace.Host, host), cmp.Or(t.Workspace.WorkspaceID, id)
		}
		p := profiles[0]
		bundleID, profileID := workspaceID(host, id), workspaceID(p.Host, p.WorkspaceID)
		if host != "" && canonicalHost(host) != canonicalHost(p.Host) || bundleID != "" && profileID != "" && bundleID != profileID {
			log.Debugf(ctx, "config-remote-sync telemetry skipped: profile %s is for another workspace than the bundle's %s", profileName, host)
			return ctx
		}
	}
	cfg := (&config.Workspace{Profile: profileName}).Config(ctx)
	// As for a bundle's own profile, the profile wins over auth env vars, so the
	// upload goes to the host checked above.
	cfg.Loaders = databrickscfg.ProfileAuthLoaders
	cfg.Credentials = auth.CredentialsWithoutRefresh()
	return cmdctx.SetConfigUsed(ctx, cfg)
}

// workspaceID returns the explicit workspace ID, else the one in the host URL.
func workspaceID(host, id string) string {
	return cmp.Or(id, auth.ExtractHostQueryParams(host).WorkspaceID)
}

func canonicalHost(host string) string {
	return (&sdkconfig.Config{Host: host}).CanonicalHostName()
}

// selectedTarget returns the target loading the bundle selects: the named one,
// else DATABRICKS_BUNDLE_TARGET, else the only target or the one marked default.
func selectedTarget(ctx context.Context, b *bundle.Bundle, name string) string {
	if name == "" {
		name, _ = bundleenv.Target(ctx)
	}
	if name != "" {
		return name
	}
	var defaults []string
	for n, t := range b.Config.Targets {
		if len(b.Config.Targets) == 1 || (t != nil && config.IsExplicitlyEnabled(t.Default)) {
			defaults = append(defaults, n)
		}
	}
	if len(defaults) != 1 {
		return ""
	}
	return defaults[0]
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
			Save:                               s.Save,
			Engine:                             string(s.Engine),
			ChangesTotal:                       s.ChangesTotal,
			AddCount:                           s.AddCount,
			ReplaceCount:                       s.ReplaceCount,
			RemoveCount:                        s.RemoveCount,
			RecreateForcingChanges:             s.RecreateForcingChanges,
			OverwrittenLocalEdits:              s.OverwrittenLocalEdits,
			ResourceChanges:                    resourceChanges,
			FilesChangedCount:                  s.FilesChangedCount,
			FilesWrittenCount:                  s.FilesWrittenCount,
			SkippedChangesCount:                s.SkippedChangesCount,
			ClusterPolicyNotFoundCount:         s.ClusterPolicyNotFoundCount,
			ClusterPolicyPermissionDeniedCount: s.ClusterPolicyPermissionDeniedCount,
			ClusterPolicyReadFailedCount:       s.ClusterPolicyReadFailedCount,
			RefsRetargeted:                     s.Restore.Retargeted,
			RefsFromSiblings:                   s.Restore.FromSiblings,
			StateSerial:                        s.StateSerial,
			StateLineage:                       s.StateLineage,
			StateSource:                        s.StateSource,
			StatesAvailableCount:               s.StatesAvailableCount,
			StateResourceIDs:                   s.StateResourceIDs,
			SelectedResourceIDs:                s.SelectedResourceIDs,
			InGitRepo:                          s.InGitRepo,
			ErrorMessage:                       s.ErrorMessage,
			ErrorCategory:                      s.ErrorCategory,
		},
	})
}
