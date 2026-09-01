package autotest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	bundleconfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
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

	// unique is the suffix this harness gave its resource, reused for the values of identity
	// fields so no two runs against one workspace ask for the same name.
	unique string
}

func newClient(t *testing.T) *databricks.WorkspaceClient {
	if isCloud() {
		w, err := databricks.NewWorkspaceClient()
		require.NoError(t, err)
		return w
	}
	server := testserver.New(tolerantT{t})
	testserver.AddDefaultHandlers(server)
	// This suite performs thousands of updates, and an asynchronous resource that reports
	// itself in-progress once costs a full second of SDK backoff each time -- serving
	// endpoints alone accounted for more wall time than every other resource type
	// combined. Waiter behaviour is covered by the acceptance suite, which leaves the
	// simulation on.
	server.SettleAsyncImmediately()
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
// uniqueNameVar names the run's own suffix. Every fixture uses it for the resource name, and it
// is the one variable a value library cannot expand for itself.
const uniqueNameVar = "UNIQUE_NAME"

func templateVars(uniqueName, userName string) map[string]string {
	vars := map[string]string{
		uniqueNameVar: uniqueName,
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
		if key == uniqueNameVar || key == "CURRENT_USER_NAME" {
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
func newHarness(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, resourceType, uniqueName string, base any, deps map[string]any, variables map[string]string) (*bundleHarness, error) {
	dir := t.TempDir()
	if err := copyDir(dataDir, dir); err != nil {
		return nil, err
	}

	rememberWorkspaceUser(user.UserName)
	yml, err := renderBundle(resourceType, uniqueName, user.UserName, base, deps, variables)
	if err != nil {
		return nil, err
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
	for name, value := range variables {
		ctx = env.Set(ctx, "BUNDLE_VAR_"+name, value)
	}
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
		return nil, fmt.Errorf("loading %s: %s", resourceType, firstError(diags))
	}

	// Pin the user so PopulateCurrentUser makes no API call: a harness is built per
	// config and again on every rebuild, which on cloud would be thousands of identical
	// CurrentUser.Me requests.
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &bundleconfig.User{User: user} //exhaustruct:ignore
	})

	phases.Initialize(ctx, b)
	if diags := logdiag.FlushCollected(ctx); diags.HasError() {
		return nil, fmt.Errorf("initializing %s: %s", resourceType, firstError(diags))
	}

	// Drop the sub-resource blocks before anything is planned. They are separate plan
	// nodes describing an ACL rather than the resource, and they are out of scope here.
	//
	// Leaving them in also mis-attributes a known bug: a recreate re-keys the parent but
	// not its child's state entry, so the child re-proposes the same update forever --
	// which would be reported against whichever field happened to trigger the recreate,
	// once per field. That bug has its own coverage in
	// acceptance/bundle/resources/volumes/recreate.
	if err := stripSubResources(&b.Config, ctx); err != nil {
		return nil, err
	}

	node, err := resourceNode(&b.Config, resourceType)
	if err != nil {
		return nil, err
	}

	// A real deploy runs deploy.ResourcePathMkdir before creating resources, because an
	// alert or dashboard is created inside ${workspace.resource_path} and the backend 404s
	// on a missing parent. This suite plans and applies directly, so it has to do the same
	// step itself -- otherwise the whole type reports one BASE_ERROR that says nothing about
	// any field.
	bundle.ApplySeq(ctx, b, deploy.ResourcePathMkdir())
	if diags := logdiag.FlushCollected(ctx); diags.HasError() {
		return nil, fmt.Errorf("creating the resource path for %s: %s", resourceType, firstError(diags))
	}

	harness := &bundleHarness{
		t:         t,
		ctx:       ctx,
		client:    client,
		bundle:    b,
		node:      node,
		statePath: filepath.Join(dir, "resources.json"),
		unique:    uniqueName,
	}

	return harness, nil
}

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
		// A pendingApply even here: every caller cancels what plan hands back, and a nil one
		// would turn a state-file problem into a panic.
		return &pendingApply{ctx: ctx, cancel: cancel, db: db}, nil, diag.FromErr(err)
	}
	plan, err := db.CalculatePlan(ctx, h.client, &h.bundle.Config)
	diags := logdiag.FlushCollected(ctx)
	if err != nil && !diags.HasError() {
		diags = append(diags, diag.FromErr(err)...)
	}
	return &pendingApply{ctx: ctx, cancel: cancel, db: db}, plan, diags
}

// readPlan is plan for a caller that will not apply: the operation context is released
// straight away rather than left holding a ten-minute timer until the test ends.
func (h *bundleHarness) readPlan() (*deployplan.Plan, diag.Diagnostics) {
	pending, plan, diags := h.plan()
	pending.cancel()
	return plan, diags
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
		// Finalize even here: the upgrade can fail with a WAL already created, and leaving it
		// behind makes every later operation fail with an unexpected-WAL error instead.
		_, _ = db.StateDB.Finalize(ctx)
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
		pending.cancel()
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

// edit runs fn over the typed resource under test and syncs the result back into the
// dynamic tree, which is what the planner reads (config.Root.GetResourceConfig). Enter
// and exit around a typed edit is the same contract every bundle mutator follows.
func (h *bundleHarness) edit(fn func(resource any) error) error {
	if err := h.bundle.Config.MarkMutatorEntry(h.ctx); err != nil {
		return err
	}
	resource, err := structaccess.GetByString(&h.bundle.Config, h.node)
	if err == nil {
		err = fn(resource)
	}
	if exitErr := h.bundle.Config.MarkMutatorExit(h.ctx); exitErr != nil {
		return exitErr
	}
	return err
}

// resource returns the typed resource under test, e.g. *resources.Schema.
func (h *bundleHarness) resource() (any, error) {
	return structaccess.GetByString(&h.bundle.Config, h.node)
}

// depKey is the bundle key a dependency is declared under: its own resource type, singularised
// only in the sense that it is unique per type, so base references read ${resources.<type>.<type>...}.
func depKey(depType string) string {
	return depType
}

// resourceKey is the name every fixture's resource is declared under. One resource per bundle,
// so the name carries nothing and only has to be stable.
const resourceKey = "foo"

// renderBundle writes the fixture's base out as a bundle. The value library holds the whole
// resource, so this is the only place a databricks.yml comes from: the base is marshalled back
// to YAML, indented under its resource type, and given the bundle block.
//
// $UNIQUE_NAME survives loadFieldValues unexpanded and is expanded here, because it belongs to
// one deploy: a rebuild gets a new name, while a value library is read once for the whole run.
func renderBundle(resourceType, uniqueName, userName string, base any, deps map[string]any, variables map[string]string) (string, error) {
	// Marshalled as one document rather than spliced together as indented text: re-indenting
	// someone else's YAML has to reason about block scalars, where a blank line is content.
	resources := map[string]any{resourceType: map[string]any{resourceKey: base}}
	// A dependency is declared under its own type with its own key, so base can reference it as
	// ${resources.<type>.<key>.<field>} the way a bundle normally would. The key is the type's
	// own name, which keeps a reference readable and cannot collide with the resource under test.
	for depType, body := range deps {
		if depType == resourceType {
			return "", fmt.Errorf("dep %s is the resource type under test", depType)
		}
		resources[depType] = map[string]any{depKey(depType): body}
	}

	document := map[string]any{
		"bundle":    map[string]any{"name": "test-bundle-$UNIQUE_NAME"},
		"resources": resources,
	}
	// Declared without a value: the value comes from the environment below, since a default here
	// would be resolved before the config-format validations run.
	if len(variables) > 0 {
		declared := map[string]any{}
		for name := range variables {
			declared[name] = map[string]any{"description": "set by the field catalog"}
		}
		document["variables"] = declared
	}

	body, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}

	vars := templateVars(uniqueName, userName)
	var missing string
	yml := os.Expand(string(body), func(key string) string {
		if value, ok := vars[key]; ok {
			return value
		}
		// A bundle's own interpolation shares this syntax -- ${var.secret_value},
		// ${resources.postgres_projects.x.name} -- and belongs to the config, not to the suite.
		// Told apart by the dot: every variable the suite provides is a bare upper-case name, and
		// every bundle reference is dotted. Restored verbatim so the config still carries it.
		if strings.Contains(key, ".") {
			return "${" + key + "}"
		}
		// Expanding an unknown variable to "" turns a required field into null, and the failure
		// then surfaces as a confusing validation error much later.
		missing = key
		return ""
	})
	if missing != "" {
		return "", fmt.Errorf("%s.yml uses $%s, which this suite does not provide", resourceType, missing)
	}
	return yml, nil
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
// value is nil -- which is how "absent" is expressed, since a Go struct has no absent.
func (h *bundleHarness) setField(path string, value any) error {
	node, err := structpath.ParsePath(path)
	if err != nil {
		return err
	}
	return h.edit(func(resource any) error {
		return setNode(resource, node, substituteUnique(value, h.unique))
	})
}

// substituteUnique resolves the placeholder in an identity field's value against the suffix
// this harness's resource actually carries. A rebuild changes that suffix, and the resource it
// replaced is still alive under the old one until the test ends.
func substituteUnique(value any, unique string) any {
	text, ok := value.(string)
	if !ok || !strings.Contains(text, uniqueMarker) {
		return value
	}
	return strings.ReplaceAll(text, uniqueMarker, unique)
}

// setNode is setField without the bundle: the edit itself, for the unit tests.
func setNode(resource any, node *structpath.PathNode, value any) error {
	if value == nil {
		return removeField(resource, node)
	}
	if err := ensurePath(resource, node); err != nil {
		return err
	}
	converted, err := coerce(resource, node, value)
	if err != nil {
		return err
	}
	return structaccess.Set(resource, node, converted)
}

// removeField makes a field absent. For a struct field that means the zero value with the
// field dropped from ForceSendFields, which is what structaccess.Set does with nil; a map
// entry and a slice element have to go instead of being zeroed.
func removeField(resource any, node *structpath.PathNode) error {
	parent := node.Parent()
	// An error here means the path does not lead anywhere in this config -- a nil object on
	// the way down -- which is the same conclusion as an absent parent: nothing to remove.
	container, err := structaccess.Get(resource, parent)
	if err != nil || isAbsent(container) {
		return nil //nolint:nilerr // an unreachable parent means the field is already absent
	}

	if index, isIndex := node.Index(); isIndex {
		return removeIndex(resource, parent, container, index)
	}
	if key, hasKey := node.StringKey(); hasKey {
		if value := reflect.ValueOf(container); value.Kind() == reflect.Map {
			value.SetMapIndex(reflect.ValueOf(key).Convert(value.Type().Key()), reflect.Value{})
			return nil
		}
	}
	return structaccess.Set(resource, node, nil)
}

// removeIndex drops the last element of a slice, which is the only index that can be made
// absent: removing any earlier one shifts its successor into the same path, so the field
// would still be there holding a different value and the transition would be recorded as a
// move from absent that never happened.
func removeIndex(resource any, parent *structpath.PathNode, container any, index int) error {
	value := reflect.ValueOf(container)
	if value.Kind() != reflect.Slice {
		return fmt.Errorf("cannot remove %s[%d]: parent is %s", parent, index, value.Kind())
	}
	if index < 0 || index >= value.Len() {
		return nil
	}
	if index != value.Len()-1 {
		return fmt.Errorf("cannot make %s[%d] absent: %s[%d] would shift into it", parent, index, parent, index+1)
	}
	// A fresh slice, not a re-slice: appending into the original backing array would also
	// change any copy of the slice the suite is holding on to.
	trimmed := reflect.MakeSlice(value.Type(), 0, index)
	trimmed = reflect.AppendSlice(trimmed, value.Slice(0, index))
	return structaccess.Set(resource, parent, trimmed.Interface())
}

// ensurePath makes the places a path passes through exist, which is what writing the same
// nested key into databricks.yml would do: an absent object is allocated, and a list too
// short for an index is grown. It works top down, so the type of each missing level comes
// from the level above it, which by then exists.
func ensurePath(resource any, node *structpath.PathNode) error {
	nodes := node.AsSlice()
	for i, prefix := range nodes {
		if index, isIndex := prefix.Index(); isIndex {
			if err := growSlice(resource, prefix.Parent(), index); err != nil {
				return err
			}
			continue
		}
		if i == len(nodes)-1 {
			// The leaf is what the caller is about to write.
			break
		}
		if _, nextIsIndex := nodes[i+1].Index(); nextIsIndex {
			// growSlice creates the list itself, at the length the index needs.
			continue
		}
		if current, err := structaccess.Get(resource, prefix); err == nil && !isAbsent(current) {
			continue
		}
		typ, err := typeAt(resource, prefix)
		if err != nil {
			return err
		}
		empty, err := emptyValue(typ)
		if err != nil {
			return fmt.Errorf("cannot create %s: %w", prefix, err)
		}
		if err := structaccess.Set(resource, prefix, empty); err != nil {
			return err
		}
	}
	return nil
}

// growSlice extends a list until an index is addressable, appending zero elements. The
// index is either the one the config already had, or the first of a list this suite is
// putting back after removing its only entry.
func growSlice(resource any, parent *structpath.PathNode, index int) error {
	container, err := structaccess.Get(resource, parent)
	if err != nil {
		return err
	}
	// An empty list reads back as absent -- omitempty hides it -- so the type comes from the
	// declaration rather than from the value.
	typ, err := typeAt(resource, parent)
	if err != nil {
		return err
	}
	if typ.Kind() != reflect.Slice {
		return fmt.Errorf("cannot index %s: %s", parent, typ.Kind())
	}

	grown := reflect.MakeSlice(typ, index+1, index+1)
	if !isAbsent(container) {
		value := reflect.ValueOf(container)
		if value.Len() > index {
			return nil
		}
		reflect.Copy(grown, value)
	}
	return structaccess.Set(resource, parent, grown.Interface())
}

// typeAt returns the declared type of the field at node. The parent has to exist, which
// ensurePath guarantees by allocating top down.
func typeAt(resource any, node *structpath.PathNode) (reflect.Type, error) {
	parent := node.Parent()
	typ := reflect.TypeOf(resource)
	if !parent.IsRoot() {
		value, err := structaccess.Get(resource, parent)
		if err != nil {
			return nil, err
		}
		if isAbsent(value) {
			return nil, fmt.Errorf("%s is absent", parent)
		}
		typ = reflect.TypeOf(value)
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if _, isIndex := node.Index(); isIndex {
		if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
			return nil, fmt.Errorf("cannot index %s", typ.Kind())
		}
		return typ.Elem(), nil
	}
	key, ok := node.StringKey()
	if !ok {
		return nil, fmt.Errorf("unsupported path %s", node)
	}
	switch typ.Kind() {
	case reflect.Map:
		return typ.Elem(), nil
	case reflect.Struct:
		field, _, ok := structaccess.FindStructFieldByKeyType(typ, key)
		if !ok {
			return nil, fmt.Errorf("field %q not found in %s", key, typ)
		}
		return field.Type, nil
	default:
		return nil, fmt.Errorf("cannot access %q on %s", key, typ.Kind())
	}
}

// coerce converts a value from the value library into the field's own type. A scalar is
// left to structaccess, which converts between the numeric and string kinds; a map or list
// has to go through the type's own JSON unmarshaller, since []any is assignable to nothing
// and an SDK struct or enum decodes itself.
func coerce(resource any, node *structpath.PathNode, value any) (any, error) {
	switch reflect.ValueOf(value).Kind() {
	case reflect.Map, reflect.Slice:
	default:
		return value, nil
	}

	typ, err := typeAt(resource, node)
	if err != nil {
		return nil, err
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if reflect.TypeOf(value).AssignableTo(typ) {
		// Already the field's own type: a container value read back off the resource.
		return value, nil
	}

	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	converted := reflect.New(typ)
	if err := json.Unmarshal(body, converted.Interface()); err != nil {
		return nil, fmt.Errorf("cannot use %v as %s: %w", value, typ, err)
	}
	return converted.Elem().Interface(), nil
}

// emptyValue builds the empty container to put at an absent level of a path.
func emptyValue(typ reflect.Type) (any, error) {
	switch typ.Kind() {
	case reflect.Pointer:
		return reflect.New(typ.Elem()).Interface(), nil
	case reflect.Map:
		return reflect.MakeMap(typ).Interface(), nil
	case reflect.Slice:
		return reflect.MakeSlice(typ, 0, 0).Interface(), nil
	case reflect.Struct:
		return reflect.New(typ).Elem().Interface(), nil
	default:
		return nil, fmt.Errorf("%s is not a container", typ)
	}
}

// isAbsent reports whether a value read out of the config is not there at all. A nil
// pointer or map reads back as a typed nil, which is not a nil interface.
func isAbsent(value any) bool {
	if value == nil {
		return true
	}
	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// snapshot serializes the resource under test, so a field can be put back to what the
// base config had once its transitions are done. JSON is the same representation the API
// uses, and the SDK's own unmarshaller restores ForceSendFields, so an absent field comes
// back absent rather than as an explicit empty value.
func (h *bundleHarness) snapshot() []byte {
	resource, err := h.resource()
	require.NoError(h.t, err)
	body, err := json.Marshal(resource)
	require.NoError(h.t, err)
	return body
}

// restore puts the resource back to a snapshot taken earlier.
func (h *bundleHarness) restore(snapshot []byte) {
	_ = h.edit(func(resource any) error {
		value := reflect.ValueOf(resource).Elem()
		value.Set(reflect.Zero(value.Type()))
		return json.Unmarshal(snapshot, resource)
	})
}

// subResourceKinds are the child plan nodes this suite removes from every config; see
// stripSubResources.
var subResourceKinds = []string{"permissions", "grants"}

// stripSubResources drops the permissions and grants blocks from every resource of an
// initialized config.
func stripSubResources(root *bundleconfig.Root, ctx context.Context) error {
	if err := root.MarkMutatorEntry(ctx); err != nil {
		return err
	}
	err := forEachResource(root, func(_ string, resource any) error {
		typ := reflect.TypeOf(resource)
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		for _, kind := range subResourceKinds {
			if _, _, ok := structaccess.FindStructFieldByKeyType(typ, kind); !ok {
				continue
			}
			if err := structaccess.SetByString(resource, kind, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if exitErr := root.MarkMutatorExit(ctx); exitErr != nil {
		return exitErr
	}
	return err
}

// forEachResource visits every declared resource with its config node key.
func forEachResource(root *bundleconfig.Root, fn func(node string, resource any) error) error {
	for _, group := range root.Resources.AllResources() {
		for key, resource := range group.Resources {
			node := "resources." + group.Description.PluralName + "." + key
			if err := fn(node, resource); err != nil {
				return fmt.Errorf("%s: %w", node, err)
			}
		}
	}
	return nil
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
	plan, diags := h.readPlan()
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
