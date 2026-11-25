package dresources

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	_, client := setupTestServerClient(t)

	for resourceType, resource := range SupportedResources() {
		t.Run(resourceType, func(t *testing.T) {
			adapter, err := NewAdapter(resource, resourceType, client)
			require.NoError(t, err)
			require.NotNil(t, adapter)

			testCRUD(t, resourceType, adapter, client)
		})
	}

	m, err := InitAll(client)
	require.NoError(t, err)
	require.Len(t, m, len(SupportedResources()))
}

// testIgnoreFilter encapsulates the logic for filtering fields based on ignore_remote_changes config.
type testIgnoreFilter struct {
	ignoreFields   map[string]bool
	ignorePatterns []*structpath.PatternNode
	adapter        *Adapter
}

// newTestIgnoreFilter creates a filter from the adapter's resource configs.
// It also ignores fields that exist in StateType but not in RemoteType, because
// those are automatically suppressed by the planner (reason: missing_in_remote)
// and RemapState cannot populate them from remote state.
func newTestIgnoreFilter(adapter *Adapter, group string) *testIgnoreFilter {
	ignoreFields := make(map[string]bool)
	var ignorePatterns []*structpath.PatternNode
	for _, cfg := range []*ResourceLifecycleConfig{adapter.ResourceConfig(), adapter.GeneratedResourceConfig()} {
		if cfg == nil {
			continue
		}
		for _, p := range cfg.IgnoreRemoteChanges {
			ignoreFields[p.Field.String()] = true
			if group == "materialization_sets" {
				ignorePatterns = append(ignorePatterns, p.Field)
			}
		}
	}
	// Auto-include fields present in StateType but absent from RemoteType.
	_ = structwalk.WalkType(adapter.StateType(), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
		if path.IsRoot() {
			return true
		}
		if structaccess.ValidatePattern(adapter.RemoteType(), path) != nil {
			ignoreFields[path.String()] = true
			return false
		}
		return true
	})
	return &testIgnoreFilter{ignoreFields: ignoreFields, ignorePatterns: ignorePatterns, adapter: adapter}
}

// shouldIgnore returns true if the field at the given path should be ignored.
func (f *testIgnoreFilter) shouldIgnore(path string) bool {
	if f.ignoreFields[path] {
		return true
	}
	// Check if this is a nested field under an ignored top-level field
	topLevelField := path
	if prefix, _, ok := strings.Cut(path, "."); ok {
		topLevelField = prefix
	}

	parts := strings.Split(topLevelField, "[")
	if len(parts) > 1 {
		topLevelField = parts[0]
	}

	if f.ignoreFields[topLevelField] {
		return true
	}

	// Pattern rules are collected for materialization_sets alone, so this loop is empty for every
	// other resource. A rule inside a slice (foo[*].bar) never matches the indexed path a walk
	// produces (foo[0].bar); jobs, apps and database_instances carry such rules too, so widening
	// the comparison to them is its own change.
	parsed, err := structpath.ParsePath(path)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(f.ignorePatterns, parsed.HasPatternPrefix)
}

// filterChanges returns only the changes that should not be ignored.
// It also filters out the "updated_at" timestamp field.
func (f *testIgnoreFilter) filterChanges(changes []structdiff.Change) []structdiff.Change {
	var relevantChanges []structdiff.Change
	for _, change := range changes {
		fieldName := change.Path.String()
		if !f.shouldIgnore(fieldName) {
			relevantChanges = append(relevantChanges, change)
		}
	}
	return relevantChanges
}

// requireEqual compares two structs and fails the test if there are differences
// that are not in the ignore_remote_changes list.
func (f *testIgnoreFilter) requireEqual(t *testing.T, expected, actual any, msgAndArgs ...any) {
	changes, err := structdiff.GetStructDiff(expected, actual, nil)
	require.NoError(t, err)
	relevantChanges := f.filterChanges(changes)
	require.Empty(t, relevantChanges, msgAndArgs...)
}

// prepareTestState creates the resources group depends on and returns the state DoCreate takes,
// built from the group's fixture in testDeps or testConfig.
func prepareTestState(t *testing.T, group string, adapter *Adapter, client *databricks.WorkspaceClient) any {
	var inputConfig any
	var err error

	prepDeps, hasDeps := testDeps[group]
	if hasDeps {
		inputConfig, err = prepDeps(t.Context(), client)
		require.NoError(t, err)
	} else {
		var ok bool
		inputConfig, ok = testConfig[group]

		if ok {
			// For permissions, PrepareState accepts any, so skip strict type check
			if adapter.InputConfigType().String() != "interface {}" {
				require.Equal(t, adapter.InputConfigType().String(), reflect.TypeOf(inputConfig).String())
			}
		} else {
			inputConfig = reflect.New(adapter.InputConfigType().Elem()).Interface()
		}
	}

	newState, err := adapter.PrepareState(inputConfig)
	require.NoError(t, err, "PrepareState failed")

	// The snapshot is content-addressed: DoCreate reads a zip from disk and the server
	// returns a path derived from that zip's hash. The generic fixture has no ZipPath, so
	// stage a real zip and align RelativePath/FullPath with its hash; otherwise the
	// create→read round-trip compares a fixture path against a hash-derived remote path
	// and fails. This is inherent to content-addressed resources, not a missing hook.
	if ss, ok := newState.(*SnapshotState); ok {
		snap := inputConfig.(*resources.Snapshot)
		content := []byte("test snapshot content")
		hash := snapshot.HashFromContent(content)
		zipPath := filepath.Join(t.TempDir(), hash+".zip")
		require.NoError(t, os.WriteFile(zipPath, content, 0o600))
		ss.ZipPath = filepath.ToSlash(zipPath)
		ss.RelativePath = ss.BundleID + "/" + hash
		ss.FullPath = snap.RemoteRoot + "/" + ss.RelativePath + "/" + snapshot.ContentSubdir
	}

	return newState
}

func testCRUD(t *testing.T, group string, adapter *Adapter, client *databricks.WorkspaceClient) {
	newState := prepareTestState(t, group, adapter, client)

	ctx := t.Context()

	// initial DoRead() cannot find the resource
	remote, err := adapter.DoRead(ctx, "1234")
	require.Nil(t, remote)
	require.Error(t, err)
	// TODO: if errors.Is(err, databricks.ErrResourceDoesNotExist) {... }

	createdID, remoteStateFromCreate, err := adapter.DoCreate(ctx, newState)
	require.NoError(t, err, "DoCreate failed state=%v", newState)
	require.NotEmpty(t, createdID, "ID returned from DoCreate was empty")

	remote, err = adapter.DoRead(ctx, createdID)
	require.NoError(t, err)
	require.NotNil(t, remote)

	// RemoteType is included verbatim in the JSON plan's "remote_state" field,
	// so it must survive a JSON round-trip without losing fields.
	assertJSONRoundTrip(t, reflect.ValueOf(remote).Elem().Interface(), "RemoteType "+group)

	remappedState, err := adapter.RemapState(remote)
	require.NoError(t, err)
	require.NotNil(t, remappedState)

	// Create filter for fields that should be ignored based on ignore_remote_changes config.
	ignoreFilter := newTestIgnoreFilter(adapter, group)

	if remoteStateFromCreate != nil {
		remappedRemoteStateFromCreate, err := adapter.RemapState(remoteStateFromCreate)
		require.NoError(t, err)
		ignoreFilter.requireEqual(t, remappedState, remappedRemoteStateFromCreate,
			"unexpected differences between remappedState and remappedRemoteStateFromCreate")
	}

	remoteStateFromWaitCreate, err := adapter.WaitAfterCreate(ctx, createdID, newState)
	require.NoError(t, err)
	if remoteStateFromWaitCreate != nil {
		// Re-read after the wait: the earlier remote can be stale for resources
		// whose state settles while WaitAfterCreate blocks (e.g. job_runs).
		remote, err = adapter.DoRead(ctx, createdID)
		require.NoError(t, err)
		require.Equal(t, remote, remoteStateFromWaitCreate)

		remappedState, err = adapter.RemapState(remote)
		require.NoError(t, err)
	}

	if adapter.HasDoUpdate() {
		remoteStateFromUpdate, err := adapter.DoUpdate(ctx, createdID, newState, &deployplan.PlanEntry{})
		require.NoError(t, err, "DoUpdate failed")
		if remoteStateFromUpdate != nil {
			remappedStateFromUpdate, err := adapter.RemapState(remoteStateFromUpdate)
			require.NoError(t, err)
			// Compare DoUpdate's result against a fresh DoRead: server-generated
			// fields (e.g. etag) may change on any write, so DoUpdate's return
			// value must match what DoRead returns right after.
			remotePostUpdate, err := adapter.DoRead(ctx, createdID)
			require.NoError(t, err)
			remappedPostUpdate, err := adapter.RemapState(remotePostUpdate)
			require.NoError(t, err)
			ignoreFilter.requireEqual(t, remappedPostUpdate, remappedStateFromUpdate,
				"unexpected differences between remappedState and remappedStateFromUpdate")
			// DoUpdate may mutate newState in place (e.g. etag), so update remappedState
			// to match the post-update server state for the field checks below.
			remappedState = remappedStateFromUpdate
		}

		remoteStateFromWaitUpdate, err := adapter.WaitAfterUpdate(ctx, createdID, newState)
		require.NoError(t, err)
		if remoteStateFromWaitUpdate != nil {
			remappedStateFromWaitUpdate, err := adapter.RemapState(remoteStateFromWaitUpdate)
			require.NoError(t, err)
			ignoreFilter.requireEqual(t, remappedState, remappedStateFromWaitUpdate,
				"unexpected differences between remappedState and remappedStateFromWaitUpdate")
		}
	}

	require.NoError(t, structwalk.Walk(newState, func(path *structpath.PathNode, val any, field *reflect.StructField) {
		// Skip fields configured in ignore_remote_changes.
		if ignoreFilter.shouldIgnore(path.String()) {
			return
		}
		remoteValue, err := structaccess.Get(remappedState, path)
		if err != nil {
			t.Errorf("Failed to read %s from remapped remote state %#v", path.String(), remappedState)
		}
		if val == nil {
			// t.Logf("Ignoring %s nil, remoteValue=%#v", path.String(), remoteValue)
			return
		}
		v := reflect.ValueOf(val)
		if v.IsZero() {
			// t.Logf("Ignoring %s zero (%#v), remoteValue=%#v", path.String(), val, remoteValue)
			// testserver can set field to backend-generated value
			return
		}
		// t.Logf("Testing %s v=%#v, remoteValue=%#v", path.String(), val, remoteValue)
		// We expect fields set explicitly to be preserved by testserver, which is true for all resources as of today.
		// If not true for your resource, add exception here:
		assert.Equal(t, val, remoteValue, "path=%q\nnewState=%s\nremappedState=%s", path.String(), jsonDump(newState), jsonDump(remappedState))
	}))

	err = adapter.DoDelete(ctx, createdID, newState)
	require.NoError(t, err)

	// WaitAfterDelete polls until the resource reads back gone; a NotFound is that
	// success and the caller (DeploymentUnit.waitDeleted) maps it to nil, so accept it here too.
	err = adapter.WaitAfterDelete(ctx, createdID)
	if !apierr.IsMissing(err) {
		require.NoError(t, err)
	}

	p, err := structpath.ParsePath("name")
	require.NoError(t, err)

	if adapter.HasOverrideChangeDesc() {
		err = adapter.OverrideChangeDesc(ctx, p, &deployplan.ChangeDesc{}, nil)
		require.NoError(t, err)
	}

	// A resource that implements no DoDelete (permissions, grants, job_runs)
	// leaves the resource in place, so DoRead still succeeds afterwards.
	// postgres_snapshot_schedules does implement DoDelete but has no delete
	// endpoint: it disables the schedule by setting an empty cadence set, and the
	// schedule remains readable (it is intrinsic to the branch).
	deleteIsNoop := !adapter.HasDoDelete() || group == "postgres_snapshot_schedules"
	isImmutable := strings.HasSuffix(group, "internal_immutable_snapshots")
	// Apps DoDelete is fire-and-forget: the API returns success while the app
	// sits in DELETING state for up to ~20 minutes before the record is removed.
	// A GET on the DELETING app returns the app, not 404 -- the testserver
	// mirrors that in libs/testserver/apps.go. DoRead therefore still succeeds
	// here; the planner treats this transient state as gone via ResourceApp.IsGone
	// so delete/destroy stay idempotent (acceptance/bundle/invariant/{delete,destroy}_idempotent).
	deleteLeavesDeleting := group == "apps"

	remoteAfterDelete, err := adapter.DoRead(ctx, createdID)
	switch {
	case isImmutable:
		require.NoError(t, err)
		assert.True(t, adapter.IsGone(remoteAfterDelete))
	case deleteIsNoop:
		require.NoError(t, err)
		// The resource genuinely still exists, so it must not report as gone.
		assert.False(t, adapter.IsGone(remoteAfterDelete))
	case deleteLeavesDeleting:
		require.NoError(t, err)
		require.NotNil(t, remoteAfterDelete)
		// IsGone lets the planner short-circuit the second delete on this
		// transient DELETING state; this is what keeps the delete/destroy
		// invariant tests idempotent, so assert the contract directly.
		assert.True(t, adapter.IsGone(remoteAfterDelete))
	default:
		require.Error(t, err)
		require.Nil(t, remoteAfterDelete)
	}
}

// TestResourceConfig validates that all field patterns in resource config
// exist in the corresponding StateType for each resource.
func TestResourceConfig(t *testing.T) {
	for resourceType, resource := range SupportedResources() {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		cfg := adapter.ResourceConfig()
		if cfg == nil {
			continue
		}

		t.Run(resourceType, func(t *testing.T) {
			validateResourceConfig(t, adapter.StateType(), adapter.RemoteType(), cfg)
		})
	}
}

// TestGeneratedResourceConfig validates that all field patterns in generated resource config
// exist in the corresponding StateType for each resource.
func TestGeneratedResourceConfig(t *testing.T) {
	for resourceType, resource := range SupportedResources() {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		cfg := adapter.GeneratedResourceConfig()
		if cfg == nil {
			continue
		}

		t.Run(resourceType, func(t *testing.T) {
			validateResourceConfig(t, adapter.StateType(), adapter.RemoteType(), cfg)
		})
	}
}

func validateResourceConfig(t *testing.T, stateType, remoteType reflect.Type, cfg *ResourceLifecycleConfig) {
	for _, p := range cfg.RecreateOnChanges {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "RecreateOnChanges: %s", p.Field)
	}
	for _, p := range cfg.ProvidedIDFields {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "ProvidedIDFields: %s", p.Field)
	}
	for _, p := range cfg.UpdatableIDFields {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "UpdatableIDFields: %s", p.Field)
	}
	for _, p := range cfg.ParentIDFields {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "ParentIDFields: %s", p.Field)
	}
	for _, p := range cfg.IgnoreRemoteChanges {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "IgnoreRemoteChanges: %s", p.Field)
	}
	for _, p := range cfg.BackendDefaults {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "BackendDefaults: %s", p.Field)
	}
	// stable_output_fields must be valid paths in RemoteType: the reference
	// resolver reads their value from the remote cache. The backend owns the
	// value, but the field may or may not also appear in StateType, so we only
	// require presence in RemoteType.
	for _, p := range cfg.StableOutputFields {
		assert.NoError(t, structaccess.ValidatePattern(remoteType, p.Field), "StableOutputFields %s: must be a valid RemoteType path", p.Field)
	}
}

// TestNoUpdateResourcesCoverAllFields ensures that resources which do not
// implement DoUpdate classify every settable field. Without DoUpdate the
// planner rejects any "update" action ("resource does not support update
// action but plan produced update"), so a field a user can change must be
// declared recreate_on_changes (or ignore_local_changes) to recreate instead.
// A provided_id_field also recreates on change (classifyIDField), so it counts
// as covered too. An output-only field is fine under ignore_remote_changes
// because the user never sets it, so it can never produce a user-driven change.
//
// The check walks the state type and fails on any uncovered scalar leaf. This
// keeps the recreate_on_changes lists exhaustive as the SDK adds fields.
func TestNoUpdateResourcesCoverAllFields(t *testing.T) {
	for resourceType, resource := range SupportedResources() {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)
		if adapter.HasDoUpdate() {
			continue
		}
		if adapter.HasOverrideChangeDesc() {
			// OverrideChangeDesc classifies changes programmatically (e.g.
			// vector_search_indexes handles endpoint_uuid there), which this
			// static walk can't see into.
			continue
		}

		// A user change is only neutralized by recreate_on_changes,
		// provided_id_fields, or ignore_local_changes; output-only fields are
		// covered by ignore_remote_changes since the user never sets them. A
		// root rule (omitted field) records the empty path "", which stops the
		// walk below at the root node and thus covers every field at once.
		covered := map[string]bool{}
		for _, cfg := range []*ResourceLifecycleConfig{adapter.ResourceConfig(), adapter.GeneratedResourceConfig()} {
			if cfg == nil {
				continue
			}
			for _, r := range cfg.RecreateOnChanges {
				covered[r.Field.String()] = true
			}
			for _, r := range cfg.ProvidedIDFields {
				covered[r.Field.String()] = true
			}
			for _, r := range cfg.IgnoreLocalChanges {
				covered[r.Field.String()] = true
			}
			for _, r := range cfg.IgnoreRemoteChanges {
				if strings.HasSuffix(r.Reason, "output_only") {
					covered[r.Field.String()] = true
				}
			}
		}

		t.Run(resourceType, func(t *testing.T) {
			err := structwalk.WalkType(adapter.StateType(), func(path *structpath.PatternNode, typ reflect.Type, _ *reflect.StructField) bool {
				if covered[path.String()] {
					// This field (or its enclosing object) is classified; the
					// whole subtree is covered, so stop descending.
					return false
				}
				for typ.Kind() == reflect.Pointer {
					typ = typ.Elem()
				}
				switch typ.Kind() {
				case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
					// Intermediate node: descend and check its children.
					return true
				default:
					t.Errorf("field %q is not classified; a change to it would plan an unsupported update. Add it to recreate_on_changes.", path)
					return false
				}
			})
			require.NoError(t, err)
		})
	}
}

// stableNameOptOut is the explicit escape hatch for TestNameStableOrOptedOut: a
// resource whose `name` is backend-owned (so not auto-exempt) yet must still NOT
// be declared stable_output_fields — e.g. the backend recomputes the name on
// update, so its remote value is not stable. Empty today.
var stableNameOptOut = map[string]string{}

// nameIsUserProvided reports whether `name` is a plain user-set field: present in
// StateType and touched by no ignore rule (so not output_only or otherwise
// backend-owned). Such a name resolves from local config, so a ${...name}
// reference is never delayed — and it must not be pinned stable, since a rename
// must change the reference.
func nameIsUserProvided(adapter *Adapter, path *structpath.PathNode) bool {
	if structaccess.ValidatePath(adapter.StateType(), path) != nil {
		return false
	}
	for _, cfg := range []*ResourceLifecycleConfig{adapter.ResourceConfig(), adapter.GeneratedResourceConfig()} {
		if cfg == nil {
			continue
		}
		for _, rules := range [][]FieldRule{cfg.IgnoreRemoteChanges, cfg.IgnoreLocalChanges} {
			for _, r := range rules {
				if path.HasPatternPrefix(r.Field) {
					return false
				}
			}
		}
	}
	return true
}

// TestNameStableOrOptedOut guards against silently reintroducing ES-2202624: a
// ${resources.X.name} reference delayed during X's in-place update recreates
// every dependent that uses it as an immutable parent. To prevent that, a
// backend-assigned `name` that can be referenced must be declared under
// stable_output_fields so the reference resolves from the remote cache.
//
// For every resource that supports in-place update and has a `name` in RemoteType,
// the name must be either auto-exempt (a plain user-provided field, which resolves
// from local config), declared stable, or listed in stableNameOptOut. Resources
// with no DoUpdate are skipped: they never take an in-place-update action, so the
// reference is never delayed.
func TestNameStableOrOptedOut(t *testing.T) {
	namePath := structpath.MustParsePath("name")
	for resourceType, resource := range SupportedResources() {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		if !adapter.HasDoUpdate() {
			continue
		}
		if structaccess.ValidatePath(adapter.RemoteType(), namePath) != nil {
			continue
		}

		if nameIsUserProvided(adapter, namePath) {
			t.Logf("%s: auto-exempt — name is user-provided (in StateType, not output_only or ignored); resolves from local config", resourceType)
			continue
		}

		if reason, ok := stableNameOptOut[resourceType]; ok {
			t.Logf("%s: exempt from the stable-name check: %s", resourceType, reason)
			continue
		}

		t.Run(resourceType, func(t *testing.T) {
			assert.True(t, adapter.FieldIsStableOutput(namePath),
				"backend-assigned `name` must be declared under stable_output_fields in configs/%[1]s.yml, or listed in stableNameOptOut; otherwise ${resources.%[1]s.<key>.name} references recreate dependents on an in-place update (ES-2202624)", resourceType)
		})
	}
}

func setupTestServerClient(t *testing.T) (*testserver.Server, *databricks.WorkspaceClient) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:               server.URL,
		Token:              "testtoken",
		RateLimitPerSecond: math.MaxInt,
	})
	require.NoError(t, err)
	return server, client
}

func jsonDump(obj any) string {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
