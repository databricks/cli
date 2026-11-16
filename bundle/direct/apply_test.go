package direct

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
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

	err = d.UpdateWithID(t.Context(), db, "old-id", &emptyIDState{Name: "bar"}, nil)
	require.EqualError(t, err, "update returned an empty resource id")

	assert.Equal(t, "old-id", db.GetResourceID(d.ResourceKey))
}
