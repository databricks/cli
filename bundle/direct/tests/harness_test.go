package tests

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	bundleconfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// bundleHarness drives the direct engine over one bundle config in-process. Nothing
// here shells out to the CLI or uploads files: the point of this suite is to exercise
// plan and apply for thousands of field permutations, and a `bundle deploy` per
// permutation would be dominated by bundle-file sync.
type bundleHarness struct {
	t         *testing.T
	ctx       context.Context
	client    *databricks.WorkspaceClient
	bundle    *bundle.Bundle
	node      string // "resources.schemas.foo"
	statePath string
}

func newClient(t *testing.T) *databricks.WorkspaceClient {
	if isCloud() {
		w, err := databricks.NewWorkspaceClient()
		require.NoError(t, err)
		return w
	}
	server := testserver.New(tolerantT{t})
	testserver.AddDefaultHandlers(server)
	//exhaustruct:ignore // an SDK config needs only these three fields to reach a fake server
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:               server.URL,
		Token:              "testtoken",
		RateLimitPerSecond: math.MaxInt,
	})
	require.NoError(t, err)
	return w
}

func isCloud() bool { return os.Getenv("CLOUD_ENV") != "" }

// workspaceUser is resolved once per resource type and handed to every harness, so
// PopulateCurrentUser never reaches the API. Locally the name is pinned, which keeps
// workspace paths stable between runs.
func workspaceUser(t *testing.T, client *databricks.WorkspaceClient) *iam.User {
	if !isCloud() {
		return &iam.User{UserName: testUserName} //exhaustruct:ignore
	}
	user, err := client.CurrentUser.Me(t.Context(), iam.MeRequest{}) //exhaustruct:ignore
	require.NoError(t, err)
	return user
}

// testUserName is pinned locally so workspace paths are stable between runs.
const testUserName = "tester@databricks.com"

// templateVars mirrors the variables acceptance/acceptance_test.go exports to the
// invariant configs. On cloud the harness that launched us provides the real ids.
func templateVars(uniqueName, userName string) map[string]string {
	vars := map[string]string{
		"UNIQUE_NAME": uniqueName,
		// Matches defaultSparkVersion in acceptance/acceptance_test.go.
		"DEFAULT_SPARK_VERSION":     "13.3.x-snapshot-scala2.12",
		"NODE_TYPE_ID":              nodeTypeID(),
		"CURRENT_USER_NAME":         userName,
		"TEST_DEFAULT_WAREHOUSE_ID": testserver.TestDefaultWarehouseId,
		"TEST_INSTANCE_POOL_ID":     testserver.TestDefaultInstancePoolId,
	}
	if !isCloud() {
		return vars
	}
	for key := range vars {
		// UNIQUE_NAME is this run's own, and CURRENT_USER_NAME is already the resolved
		// workspace user, which is more reliable than an env var that may not be set.
		if key == "UNIQUE_NAME" || key == "CURRENT_USER_NAME" {
			continue
		}
		if value := os.Getenv(key); value != "" {
			vars[key] = value
		}
	}
	return vars
}

// nodeTypeID mirrors getNodeTypeID in acceptance/acceptance_test.go.
func nodeTypeID() string {
	switch cloudName() {
	case "azure":
		return "Standard_E4ds_v5"
	case "gcp":
		return "n1-standard-4"
	default:
		return "i3.xlarge"
	}
}

func retryIntervalMs() string {
	if isCloud() {
		return "2000"
	}
	return "10"
}

// maxWaitSeconds caps the per-resource readiness wait. The testserver answers
// immediately, so anything non-zero there only guards against a polling loop.
func maxWaitSeconds() string {
	if isCloud() {
		return "300"
	}
	// Against the fake server every legitimate wait finishes at once, so this cap only
	// bounds the waits that cannot succeed -- an app delete, for one, which the fake
	// server holds in DELETING the way the real API does.
	return "2"
}

// newHarness renders one invariant config into a temp dir and runs the bundle
// initialize phase over it, leaving a fully-resolved config.Root ready to plan.
// newHarness returns an error rather than failing the test: a harness that cannot be
// built is usually the environment (an expired token, a workspace that rejects the
// config), and the run is more useful if that lands in the report as one bad verdict
// than if it aborts thousands of pending observations.
func newHarness(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, configName, uniqueName string) (*bundleHarness, error) {
	dir := t.TempDir()
	if err := copyDir(dataDir, dir); err != nil {
		return nil, err
	}

	src, err := os.ReadFile(filepath.Join(configsDir, configName))
	if err != nil {
		return nil, err
	}
	vars := templateVars(uniqueName, user.UserName)
	var missing string
	yml := os.Expand(string(src), func(key string) string {
		value, ok := vars[key]
		if !ok {
			// Expanding an unknown variable to "" turns a required field into null, and
			// the failure then surfaces as a confusing validation error much later.
			missing = key
		}
		return value
	})
	if missing != "" {
		return nil, fmt.Errorf("%s uses $%s, which this suite does not provide", configName, missing)
	}
	if err := os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(yml), 0o600); err != nil {
		return nil, err
	}

	// WithoutCancel because a harness outlives the scope that created it in two ways,
	// and t.Context() is cancelled at the end of both: a harness rebuilt inside a
	// field-level subtest is reused by later fields, and every harness is destroyed from
	// t.Cleanup, which runs *after* t.Context() is cancelled. Either way the SDK rate
	// limiter refuses the call with "context canceled" -- silently leaking the resource
	// in the cleanup case. Values (dbr, logdiag, cmdio, env) still propagate; the test
	// binary's own -timeout remains the backstop.
	ctx = dbr.MockRuntime(context.WithoutCancel(ctx), dbr.Environment{}) //exhaustruct:ignore
	// Thousands of deploys run through here, so cap how long any one of them waits for
	// a resource to become ready. Without a cap a resource that never reaches its
	// terminal state stalls the whole suite instead of showing up as one bad verdict.
	ctx = env.Set(ctx, bundleenv.ResourceMaxWaitVariable, maxWaitSeconds())
	// The engine's default retry interval is 15s. Recreating a resource whose delete is
	// asynchronous (apps) retries on purpose, and at 15s a run costs hours.
	ctx = env.Set(ctx, bundleenv.RetryIntervalMsVariable, retryIntervalMs())
	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)
	ctx = cmdctx.SetWorkspaceClient(ctx, client)
	// Some applies log progress through cmdio (app deployments, for one), which panics
	// without an IO in the context. Discard it: the report is the output that matters.
	ctx = cmdio.InContext(ctx, cmdio.NewTestIO(strings.NewReader(""), io.Discard, io.Discard))

	b, err := bundle.Load(ctx, dir)
	if err != nil {
		return nil, err
	}
	b.SetWorkpaceClient(client)
	b.AutoApprove = true

	phases.LoadDefaultTarget(ctx, b)
	if diags := logdiag.FlushCollected(ctx); diags.HasError() {
		return nil, fmt.Errorf("loading %s: %s", configName, firstError(diags))
	}

	// Pin the user so PopulateCurrentUser makes no API call: a harness is built per
	// config and again on every rebuild, which on cloud would be thousands of identical
	// CurrentUser.Me requests.
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &bundleconfig.User{User: user} //exhaustruct:ignore
	})

	phases.Initialize(ctx, b)
	if diags := logdiag.FlushCollected(ctx); diags.HasError() {
		return nil, fmt.Errorf("initializing %s: %s", configName, firstError(diags))
	}

	// Drop the sub-resource blocks before anything is planned. They are separate plan
	// nodes describing an ACL rather than the resource, and they are out of scope here.
	//
	// Leaving them in also mis-attributes a known bug: a recreate re-keys the parent but
	// not its child's state entry, so the child re-proposes the same update forever --
	// which would be reported against whichever field happened to trigger the recreate,
	// once per field. That bug has its own coverage in
	// acceptance/bundle/resources/volumes/recreate.
	if err := stripSubResources(&b.Config); err != nil {
		return nil, err
	}

	node, err := soleResourceNode(&b.Config)
	if err != nil {
		return nil, err
	}

	return &bundleHarness{
		t:         t,
		ctx:       ctx,
		client:    client,
		bundle:    b,
		node:      node,
		statePath: filepath.Join(dir, "resources.json"),
	}, nil
}

// cachedUser resolves the workspace user once per process: a harness is built per
// config and again on every rebuild, so on cloud this would otherwise be thousands of
// identical requests. Locally the name is pinned, which also keeps workspace paths
// stable between runs.

// opCtx returns a context with a fresh diagnostics sink. logdiag's error flag is sticky
// -- FlushCollected empties the collected diagnostics but leaves HasError true -- and
// CalculatePlan short-circuits to a bare "planning failed" whenever that flag is set. A
// per-operation context is what keeps one failed transition from poisoning every plan
// after it.
func (h *bundleHarness) opCtx() (context.Context, func()) {
	ctx := logdiag.IsolatedContext(h.ctx)
	logdiag.SetCollect(ctx, true)
	// A deadline so one operation that cannot finish is recorded as TIMEOUT instead of
	// stalling the run. Apps are the reason: renaming one recreates it, and the delete
	// half waits for the old name to leave DELETING, which the API does not cap.
	return context.WithTimeout(ctx, opTimeout())
}

func opTimeout() time.Duration {
	if isCloud() {
		return 10 * time.Minute
	}
	return 20 * time.Second
}

// plan opens the state fresh each time: dstate.Open panics on an already-open
// receiver, and Finalize resets it, so one DeploymentBundle serves one operation.
func (h *bundleHarness) plan() (*pendingApply, *deployplan.Plan, diag.Diagnostics) {
	ctx, cancel := h.opCtx()
	db := &direct.DeploymentBundle{} //exhaustruct:ignore
	if err := db.StateDB.Open(ctx, h.statePath, dstate.WithRecovery(false), dstate.WithWrite(false)); err != nil {
		cancel()
		return nil, nil, diag.FromErr(err)
	}
	plan, err := db.CalculatePlan(ctx, h.client, &h.bundle.Config)
	diags := logdiag.FlushCollected(ctx)
	if err != nil && !diags.HasError() {
		diags = append(diags, diag.FromErr(err)...)
	}
	return &pendingApply{ctx: ctx, cancel: cancel, db: db}, plan, diags
}

// pendingApply carries a planned DeploymentBundle together with the context it was
// planned under, so the apply reports into the same isolated diagnostics sink.
type pendingApply struct {
	ctx    context.Context
	cancel func()
	db     *direct.DeploymentBundle
}

// apply consumes the DeploymentBundle returned by plan.
func (h *bundleHarness) apply(p *pendingApply, plan *deployplan.Plan) diag.Diagnostics {
	defer p.cancel()
	ctx, db := p.ctx, p.db
	if err := db.StateDB.UpgradeToWrite(); err != nil {
		return diag.FromErr(err)
	}
	if err := db.InitForApply(ctx, h.client, plan); err != nil {
		_, _ = db.StateDB.Finalize(ctx)
		return diag.FromErr(err)
	}
	db.Apply(ctx, h.client, plan)
	diags := logdiag.FlushCollected(ctx)
	if _, err := db.StateDB.Finalize(ctx); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

// deploy plans and applies, returning the planned action for the node under test.
func (h *bundleHarness) deploy() (deployplan.ActionType, diag.Diagnostics) {
	pending, plan, diags := h.plan()
	if diags.HasError() {
		return deployplan.Undefined, diags
	}
	return nodeAction(plan, h.node), h.apply(pending, plan)
}

// destroy deletes everything in state. A nil configRoot makes the planner treat
// every state entry as a delete (same call phases.Destroy makes).
func (h *bundleHarness) destroy() diag.Diagnostics {
	ctx, cancel := h.opCtx()
	db := &direct.DeploymentBundle{} //exhaustruct:ignore
	if err := db.StateDB.Open(ctx, h.statePath, dstate.WithRecovery(false), dstate.WithWrite(false)); err != nil {
		cancel()
		return diag.FromErr(err)
	}
	plan, err := db.CalculatePlan(ctx, h.client, nil)
	if err != nil {
		_, _ = db.StateDB.Finalize(ctx)
		cancel()
		return diag.FromErr(err)
	}
	return h.apply(&pendingApply{ctx: ctx, cancel: cancel, db: db}, plan)
}

func nodeAction(plan *deployplan.Plan, node string) deployplan.ActionType {
	if plan == nil {
		return deployplan.Undefined
	}
	entry, ok := plan.Plan[node]
	if !ok {
		return deployplan.Undefined
	}
	return entry.Action
}

// hasDrift reports whether any node still has work planned.
func hasDrift(plan *deployplan.Plan) bool {
	for _, entry := range plan.Plan {
		if entry.Action != deployplan.Skip {
			return true
		}
	}
	return false
}

// mutate edits the bundle's dynamic config, which is what GetResourceConfig reads.
// Editing the typed structs would not be seen by the planner.
func (h *bundleHarness) mutate(fn func(dyn.Value) (dyn.Value, error)) error {
	return h.bundle.Config.Mutate(fn)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
}

// setField writes a value into the resource under test, or removes the field when the
// value is absent. Edits go through the dynamic config because that is what the
// planner reads (config.Root.GetResourceConfig).
func (h *bundleHarness) setField(path string, value dyn.Value) error {
	full, err := dyn.NewPathFromString(h.node + "." + path)
	if err != nil {
		return err
	}
	return h.mutate(func(root dyn.Value) (dyn.Value, error) {
		if !value.IsValid() {
			return deletePath(root, full)
		}
		if err := ensureParents(&root, full); err != nil {
			return dyn.InvalidValue, err
		}
		return dyn.SetByPath(root, full, value)
	})
}

// fieldSnapshot captures the resource node as deployed, so a field can be put back to
// its base value once its transitions are done.
func (h *bundleHarness) fieldSnapshot() dyn.Value {
	node, err := dyn.NewPathFromString(h.node)
	require.NoError(h.t, err)
	v, err := dyn.GetByPath(h.bundle.Config.Value(), node)
	require.NoError(h.t, err)
	return v
}

// restoreField resets one field to whatever the base config had, including absent.
func (h *bundleHarness) restoreField(base dyn.Value, path string) {
	rel, err := dyn.NewPathFromString(path)
	if err != nil {
		return
	}
	value, err := dyn.GetByPath(base, rel)
	if err != nil {
		value = dyn.InvalidValue
	}
	_ = h.setField(path, value)
}

// ensureParents creates any missing intermediate maps so a nested field can be set,
// which is what writing the same nested key into databricks.yml would do.
func ensureParents(root *dyn.Value, p dyn.Path) error {
	for i := 1; i < len(p); i++ {
		prefix := p[:i]
		if _, err := dyn.GetByPath(*root, prefix); err == nil {
			continue
		}
		if prefix[len(prefix)-1].Key() == "" {
			// An index component: growing a sequence would invent an element the
			// config never declared.
			return fmt.Errorf("cannot create %s: parent is a sequence", prefix)
		}
		updated, err := dyn.SetByPath(*root, prefix, dyn.V(map[string]dyn.Value{}))
		if err != nil {
			return err
		}
		*root = updated
	}
	return nil
}

// subResourceKinds are the child plan nodes this suite removes from every config; see
// stripSubResources.
var subResourceKinds = []string{"permissions", "grants"}

func stripSubResources(root *bundleconfig.Root) error {
	return root.Mutate(withoutSubResources)
}

// withoutSubResources returns the config with every permissions and grants block removed.
func withoutSubResources(v dyn.Value) (dyn.Value, error) {
	for _, kind := range subResourceKinds {
		pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key(kind))

		// Collect first: deleting during the walk would mutate what it is walking.
		var paths []dyn.Path
		_, err := dyn.MapByPattern(v, pattern, func(p dyn.Path, found dyn.Value) (dyn.Value, error) {
			paths = append(paths, slices.Clone(p))
			return found, nil
		})
		if err != nil {
			return dyn.InvalidValue, err
		}

		for _, path := range paths {
			v, err = deletePath(v, path)
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
	}
	return v, nil
}

// deletePath removes a key from its parent map. dyn has no delete, so the parent is
// rebuilt without the key, preserving order.
func deletePath(root dyn.Value, p dyn.Path) (dyn.Value, error) {
	key := p[len(p)-1].Key()
	if key == "" {
		return dyn.InvalidValue, fmt.Errorf("cannot remove %s: not a map key", p)
	}
	parentPath := p[:len(p)-1]

	parent, err := dyn.GetByPath(root, parentPath)
	if dyn.IsNoSuchKeyError(err) || dyn.IsCannotTraverseNilError(err) {
		// The parent object is not in this config, so the field is already absent.
		return root, nil
	}
	if err != nil {
		return dyn.InvalidValue, err
	}
	mapping, ok := parent.AsMap()
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("cannot remove %s: parent is %s", p, parent.Kind())
	}
	if _, found := mapping.GetByString(key); !found {
		return root, nil
	}

	trimmed := dyn.NewMapping()
	for _, pair := range mapping.Pairs() {
		if k, ok := pair.Key.AsString(); ok && k == key {
			continue
		}
		k, _ := pair.Key.AsString()
		trimmed.SetLoc(k, pair.Key.Locations(), pair.Value)
	}
	return dyn.SetByPath(root, parentPath, dyn.NewValue(trimmed, parent.Locations()))
}

// uniqueName keeps cloud resources from colliding between runs and between the
// parallel subtests of one run.
func uniqueName() string {
	return "f" + strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", ""))[:20]
}

// converged deploys the current config and reports whether a following plan is clean.
func (h *bundleHarness) converged() bool {
	if _, diags := h.deploy(); diags.HasError() {
		return false
	}
	_, plan, diags := h.plan()
	return !diags.HasError() && plan != nil && !hasDrift(plan)
}

// tolerantT wraps a *testing.T for the fake server. This suite deliberately drives the
// engine with odd field values, and some of them make it issue a request the fake
// server has no route for -- an empty name lands on "GET /api/2.0/serving-endpoints/".
// The fake server reports that with Errorf, which would fail the whole run; here it is
// just another observation, so errors are downgraded to log lines.
type tolerantT struct {
	*testing.T
}

func (tolerantT) Error(args ...any)                 {}
func (tolerantT) Errorf(format string, args ...any) {}
