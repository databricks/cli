package testserver_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxLifecycle(t *testing.T) {
	api := newSandboxAPI(t)
	ctx := t.Context()
	const name = "sandboxes/happy-panda-1234"
	sb, err := api.CreateSandbox(ctx, sandbox.CreateSandboxRequest{SandboxId: "happy-panda-1234", Sandbox: sandbox.Sandbox{DisplayName: "my-project", Spec: &sandbox.SandboxSpec{Compute: &sandbox.ComputeSpec{InactivityTimeout: duration.New(15 * time.Minute)}}}})
	require.NoError(t, err)
	assert.Equal(t, name, sb.Name)
	_, err = api.CreateSandbox(ctx, sandbox.CreateSandboxRequest{SandboxId: "happy-panda-1234"})
	assert.ErrorIs(t, err, apierr.ErrResourceAlreadyExists)
	_, err = api.StartSandbox(ctx, sandbox.StartSandboxRequest{Name: name})
	assert.ErrorIs(t, err, apierr.ErrBadRequest)
	updated, err := api.UpdateSandbox(ctx, sandbox.UpdateSandboxRequest{Name: name, Sandbox: sandbox.Sandbox{DisplayName: "renamed"}, UpdateMask: fieldmask.FieldMask{Paths: []string{"display_name"}}})
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.DisplayName)
	assert.Equal(t, 15*time.Minute, updated.Spec.Compute.InactivityTimeout.AsDuration())
	_, err = api.StopSandbox(ctx, sandbox.StopSandboxRequest{Name: name})
	require.NoError(t, err)
	stopped, err := api.GetSandbox(ctx, sandbox.GetSandboxRequest{Name: name})
	require.NoError(t, err)
	assert.Equal(t, sandbox.SandboxStateSandboxStateStopped, stopped.Status.State)
	_, err = api.StartSandbox(ctx, sandbox.StartSandboxRequest{Name: name})
	require.NoError(t, err)
	running, err := api.GetSandbox(ctx, sandbox.GetSandboxRequest{Name: name})
	require.NoError(t, err)
	assert.Equal(t, sandbox.SandboxStateSandboxStateRunning, running.Status.State)
	require.NoError(t, api.DeleteSandbox(ctx, sandbox.DeleteSandboxRequest{Name: name}))
	_, err = api.GetSandbox(ctx, sandbox.GetSandboxRequest{Name: name})
	assert.ErrorIs(t, err, apierr.ErrNotFound)
	all, err := api.ListSandboxesAll(ctx, sandbox.ListSandboxesRequest{})
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestSandboxPaginationAndWorkspaceIsolation(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "workspace-a"})
	require.NoError(t, err)
	other, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "workspace-b"})
	require.NoError(t, err)
	api := w.Sandbox
	for _, id := range []string{"c-panda-1234", "a-panda-1234", "b-panda-1234"} {
		_, err := api.CreateSandbox(t.Context(), sandbox.CreateSandboxRequest{SandboxId: id})
		require.NoError(t, err)
	}
	all, err := api.ListSandboxesAll(t.Context(), sandbox.ListSandboxesRequest{PageSize: 1})
	require.NoError(t, err)
	require.Len(t, all, 3)
	assert.Equal(t, "sandboxes/a-panda-1234", all[0].Name)
	assert.Equal(t, "sandboxes/c-panda-1234", all[2].Name)
	isolated, err := other.Sandbox.ListSandboxesAll(t.Context(), sandbox.ListSandboxesRequest{})
	require.NoError(t, err)
	assert.Empty(t, isolated)
}

func TestSandboxInvalidRequests(t *testing.T) {
	api := newSandboxAPI(t)
	for _, id := range []string{"", "UPPER", "ends-", "c352101c-ef45-4e12-881c-454056ca88ad"} {
		_, err := api.CreateSandbox(t.Context(), sandbox.CreateSandboxRequest{SandboxId: id})
		assert.ErrorIs(t, err, apierr.ErrInvalidParameterValue)
	}
	_, err := api.UpdateSandbox(t.Context(), sandbox.UpdateSandboxRequest{Name: "sandboxes/missing"})
	assert.ErrorIs(t, err, apierr.ErrInvalidParameterValue)
	_, err = api.StopSandbox(t.Context(), sandbox.StopSandboxRequest{Name: "sandboxes/missing"})
	assert.ErrorIs(t, err, apierr.ErrNotFound)
	_, err = api.StartSandbox(t.Context(), sandbox.StartSandboxRequest{Name: "sandboxes/missing"})
	assert.ErrorIs(t, err, apierr.ErrNotFound)
	assert.ErrorIs(t, api.DeleteSandbox(t.Context(), sandbox.DeleteSandboxRequest{Name: "sandboxes/missing"}), apierr.ErrNotFound)
}
