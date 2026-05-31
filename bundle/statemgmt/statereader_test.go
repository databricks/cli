package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/direct/dstate"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBundleClient struct {
	sdkbundle.BundleInterface
	resources []sdkbundle.Resource
	requests  []sdkbundle.ListResourcesRequest
	err       error
}

func (c *fakeBundleClient) ListResourcesAll(_ context.Context, req sdkbundle.ListResourcesRequest) ([]sdkbundle.Resource, error) {
	c.requests = append(c.requests, req)
	if c.err != nil {
		return nil, c.err
	}
	return c.resources, nil
}

func rawState(t *testing.T, s string) *json.RawMessage {
	t.Helper()
	msg := json.RawMessage(s)
	return &msg
}

func TestDMSStateReaderPopulatesStateAndPrefixesKeys(t *testing.T) {
	client := &fakeBundleClient{
		resources: []sdkbundle.Resource{
			{ResourceKey: "jobs.foo", ResourceId: "job-1", State: rawState(t, `{"name":"foo"}`)},
			{ResourceKey: "pipelines.bar", ResourceId: "pipe-2", State: rawState(t, `{"name":"bar"}`)},
		},
	}

	var db dstate.DeploymentState
	reader := NewDMSStateReader(client, "dep-1", filepath.Join(t.TempDir(), "resources.json"))
	require.NoError(t, reader.Load(t.Context(), &db))

	require.Len(t, client.requests, 1)
	assert.Equal(t, "deployments/dep-1", client.requests[0].Parent)

	entry, ok := db.GetResourceEntry("resources.jobs.foo")
	require.True(t, ok)
	assert.Equal(t, "job-1", entry.ID)
	assert.JSONEq(t, `{"name":"foo"}`, string(entry.State))
	assert.Equal(t, "job-1", db.GetResourceID("resources.jobs.foo"))

	entry, ok = db.GetResourceEntry("resources.pipelines.bar")
	require.True(t, ok)
	assert.Equal(t, "pipe-2", entry.ID)
}

func TestDMSStateReaderEmptyList(t *testing.T) {
	client := &fakeBundleClient{}

	var db dstate.DeploymentState
	reader := NewDMSStateReader(client, "dep-1", filepath.Join(t.TempDir(), "resources.json"))
	require.NoError(t, reader.Load(t.Context(), &db))

	_, ok := db.GetResourceEntry("resources.jobs.foo")
	assert.False(t, ok)
}

func TestDMSStateReaderNilStateBecomesEmpty(t *testing.T) {
	client := &fakeBundleClient{
		resources: []sdkbundle.Resource{
			{ResourceKey: "jobs.foo", ResourceId: "job-1", State: nil},
		},
	}

	var db dstate.DeploymentState
	reader := NewDMSStateReader(client, "dep-1", filepath.Join(t.TempDir(), "resources.json"))
	require.NoError(t, reader.Load(t.Context(), &db))

	entry, ok := db.GetResourceEntry("resources.jobs.foo")
	require.True(t, ok)
	assert.Equal(t, "job-1", entry.ID)
	assert.Nil(t, entry.State)
}

func TestDMSStateReaderPropagatesListError(t *testing.T) {
	wantErr := errors.New("boom")
	client := &fakeBundleClient{err: wantErr}

	var db dstate.DeploymentState
	reader := NewDMSStateReader(client, "dep-1", filepath.Join(t.TempDir(), "resources.json"))
	err := reader.Load(t.Context(), &db)
	assert.ErrorIs(t, err, wantErr)
}

func TestFileStateReaderReadsLocalState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.json")

	seed := dstate.NewDatabase("lineage-1", 3)
	seed.State["resources.jobs.foo"] = dstate.ResourceEntry{ID: "job-1", State: json.RawMessage(`{"name":"foo"}`)}
	content, err := json.Marshal(seed)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, content, 0o600))

	var db dstate.DeploymentState
	reader := NewFileStateReader(path)
	require.NoError(t, reader.Load(t.Context(), &db))

	assert.Equal(t, "lineage-1", db.Data.Lineage)
	assert.Equal(t, 3, db.Data.Serial)

	entry, ok := db.GetResourceEntry("resources.jobs.foo")
	require.True(t, ok)
	assert.Equal(t, "job-1", entry.ID)
}

func TestFileStateReaderMissingFileIsEmptyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	var db dstate.DeploymentState
	reader := NewFileStateReader(path)
	require.NoError(t, reader.Load(t.Context(), &db))

	_, ok := db.GetResourceEntry("resources.jobs.foo")
	assert.False(t, ok)
}
