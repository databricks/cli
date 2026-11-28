package direct

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emptyIDResource is a fake resource whose create and update return an empty id.
type emptyIDResource struct{}

type emptyIDState struct {
	Name string `json:"name"`
}

func (*emptyIDResource) New(client *databricks.WorkspaceClient) any {
	return &emptyIDResource{}
}

func (*emptyIDResource) PrepareState(input *emptyIDState) *emptyIDState {
	return input
}

func (*emptyIDResource) DoRead(ctx context.Context, id string) (*emptyIDState, error) {
	return &emptyIDState{}, nil
}

func (*emptyIDResource) DoCreate(ctx context.Context, newState *emptyIDState) (string, *emptyIDState, error) {
	return "", nil, nil
}

func (*emptyIDResource) DoUpdateWithID(ctx context.Context, id string, newState *emptyIDState, entry *dresources.PlanEntry) (string, *emptyIDState, error) {
	return "", nil, nil
}

func (*emptyIDResource) OverrideChangeDesc(ctx context.Context, path *structpath.PathNode, changedesc *dresources.ChangeDesc, remoteState any) error {
	return nil
}

func newEmptyIDDeploymentState(t *testing.T) *dstate.DeploymentState {
	t.Helper()
	db := &dstate.DeploymentState{}
	path := filepath.Join(t.TempDir(), "resources.json")
	require.NoError(t, db.Open(t.Context(), path, dstate.WritingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}))
	return db
}

func finalizeEmptyIDState(t *testing.T, db *dstate.DeploymentState) {
	t.Helper()
	_, err := db.Finalize(t.Context())
	require.NoError(t, err)
}

func TestCreateFailsOnEmptyID(t *testing.T) {
	adapter, err := dresources.NewAdapter((*emptyIDResource)(nil), "fake", nil)
	require.NoError(t, err)
	db := newEmptyIDDeploymentState(t)
	defer finalizeEmptyIDState(t, db)
	d := &DeploymentUnit{ResourceKey: "resources.fakes.foo", Adapter: adapter}

	err = d.Create(t.Context(), db, &emptyIDState{Name: "foo"})
	require.EqualError(t, err, "create returned an empty resource id")

	_, ok := db.GetResourceEntry(d.ResourceKey)
	assert.False(t, ok)
}

func TestUpdateWithIDFailsOnEmptyID(t *testing.T) {
	adapter, err := dresources.NewAdapter((*emptyIDResource)(nil), "fake", nil)
	require.NoError(t, err)
	db := newEmptyIDDeploymentState(t)
	defer finalizeEmptyIDState(t, db)
	d := &DeploymentUnit{ResourceKey: "resources.fakes.foo", Adapter: adapter}
	require.NoError(t, db.SaveState(t.Context(), d.ResourceKey, "old-id", &emptyIDState{Name: "foo"}, nil))

	err = d.UpdateWithID(t.Context(), db, "old-id", &emptyIDState{Name: "bar"}, &deployplan.PlanEntry{})
	require.EqualError(t, err, "update returned an empty resource id")

	assert.Equal(t, "old-id", db.GetResourceID(d.ResourceKey))
}

// goneAfterDeleteResource is a fake resource whose delete fails with deleteErrors in turn and whose
// read reports the resource as gone.
type goneAfterDeleteResource struct {
	deleteErrors []error
	deleteCalls  int
}

// New returns the receiver so the test can set deleteErrors and read deleteCalls.
func (r *goneAfterDeleteResource) New(client *databricks.WorkspaceClient) any {
	return r
}

func (*goneAfterDeleteResource) PrepareState(input *emptyIDState) *emptyIDState {
	return input
}

func (*goneAfterDeleteResource) DoRead(ctx context.Context, id string) (*emptyIDState, error) {
	return nil, &apierr.APIError{StatusCode: http.StatusNotFound, ErrorCode: "NOT_FOUND", Message: "does not exist"}
}

func (*goneAfterDeleteResource) DoCreate(ctx context.Context, newState *emptyIDState) (string, *emptyIDState, error) {
	return "new-id", nil, nil
}

func (r *goneAfterDeleteResource) DoDelete(ctx context.Context, id string, _ *emptyIDState) error {
	r.deleteCalls++
	if r.deleteCalls > len(r.deleteErrors) {
		return fmt.Errorf("unexpected delete call %d", r.deleteCalls)
	}
	return r.deleteErrors[r.deleteCalls-1]
}

func (*goneAfterDeleteResource) OverrideChangeDesc(ctx context.Context, path *structpath.PathNode, changedesc *dresources.ChangeDesc, remoteState any) error {
	return nil
}

// On UC secrets, a retry of an applied delete failed with a 400 "Path resolution failed" instead of
// a NotFound.
var errDeleteRetry = &apierr.APIError{StatusCode: http.StatusBadRequest, ErrorCode: "INVALID_PARAMETER_VALUE", Message: "Path resolution failed"}

func testDeleteGoneDespiteError(t *testing.T, deleteErrors []error, run func(ctx context.Context, d *DeploymentUnit, db *dstate.DeploymentState) error) string {
	t.Helper()
	var logs bytes.Buffer
	ctx := log.NewContext(t.Context(), slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: log.LevelDebug})))
	ctx = env.Set(ctx, retry.IntervalMsVariable, "1")

	resource := &goneAfterDeleteResource{deleteErrors: deleteErrors}
	adapter, err := dresources.NewAdapter(resource, "fake", nil)
	require.NoError(t, err)
	db := newEmptyIDDeploymentState(t)
	defer finalizeEmptyIDState(t, db)
	d := &DeploymentUnit{ResourceKey: "resources.fakes.foo", Adapter: adapter}
	require.NoError(t, db.SaveState(ctx, d.ResourceKey, "old-id", &emptyIDState{Name: "foo"}, nil))

	require.NoError(t, run(ctx, d, db))
	assert.Equal(t, len(deleteErrors), resource.deleteCalls)
	return logs.String()
}

// A single DoDelete call that fails although the resource is gone: the SDK's own retry of an applied
// delete looks like this. Retries in libs/retry are covered by acceptance/bundle/resources/secrets/delete_transient_500.
func TestDeleteGoneDespiteErrorIsNotAWarning(t *testing.T) {
	logs := testDeleteGoneDespiteError(t, []error{errDeleteRetry}, func(ctx context.Context, d *DeploymentUnit, db *dstate.DeploymentState) error {
		return d.Delete(ctx, db, "old-id")
	})
	assert.Contains(t, logs, "level=DEBUG msg=\"Treating resources.fakes.foo id=old-id as already deleted despite delete error: Path resolution failed\"")
	assert.NotContains(t, logs, "level=WARN msg=\"Treating")
}
