package phases_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindingLocalCommitFailureCompletesVersion(t *testing.T) {
	for _, operation := range []string{"bind", "unbind"} {
		t.Run(operation, func(t *testing.T) {
			ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
			ctx = logdiag.InitContext(ctx)
			b := &bundle.Bundle{BundleRootPath: t.TempDir()}
			root, diags := config.LoadFromBytes("databricks.yml", []byte(`bundle:
  name: test
  target: default
  deployment:
    lock:
      enabled: false
experimental:
  deployment_history: true
workspace:
  root_path: /Workspace/test
  state_path: /Workspace/test/state
resources:
  jobs:
    bound:
      name: configured
`))
			require.Empty(t, diags)
			b.Config = *root
			_, statePath := b.StateFilenameDirect(ctx)
			if operation == "unbind" {
				data := dstate.NewDatabase("lineage", 0)
				data.State["resources.jobs.bound"] = dstate.ResourceEntry{ID: "20", State: json.RawMessage(`{"name":"configured"}`)}
				state, err := json.Marshal(data)
				require.NoError(t, err)
				require.NoError(t, os.MkdirAll(filepath.Dir(statePath), 0o700))
				require.NoError(t, os.WriteFile(statePath, state, 0o600))
			}

			server := testserver.New(t)
			server.Handle("GET", "/api/2.2/jobs/get", func(req testserver.Request) any {
				return jobs.Job{JobId: 20, Settings: &jobs.JobSettings{Name: "remote"}}
			})
			server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
				return map[string]any{"object_id": 123, "object_type": "FILE"}
			})
			server.Handle("GET", "/api/2.0/bundle/deployments/123", func(req testserver.Request) any {
				return bundledeployments.Deployment{Name: "deployments/123"}
			})
			server.Handle("POST", "/api/2.0/bundle/deployments/123/versions", func(req testserver.Request) any {
				// The WAL is already open. Make the atomic state replacement fail after the state write.
				require.FileExists(t, statePath+".wal")
				if operation == "unbind" {
					require.NoError(t, os.Remove(statePath))
				}
				require.NoError(t, os.Mkdir(statePath, 0o700))
				return bundledeployments.Version{Name: "deployments/123/versions/1", VersionId: "1"}
			})
			server.Handle("PATCH", "/api/2.0/bundle/deployments/123/versions/1/operations/jobs.bound", func(req testserver.Request) any {
				return bundledeployments.Operation{}
			})
			var completion bundledeployments.CompleteVersionRequest
			server.Handle("POST", "/api/2.0/bundle/deployments/123/versions/1/complete", func(req testserver.Request) any {
				require.NoError(t, json.Unmarshal(req.Body, &completion))
				return bundledeployments.Version{}
			})
			testserver.AddDefaultHandlers(server)
			client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
			require.NoError(t, err)
			b.SetWorkpaceClient(client)
			ctx = cmdctx.SetWorkspaceClient(ctx, client)

			if operation == "unbind" {
				phases.Unbind(ctx, b, "jobs", "bound", engine.EngineDirect)
			} else {
				phases.Bind(ctx, b, &phases.BindOptions{Group: "jobs", ResourceKey: "bound", ResourceId: "20", AutoApprove: true}, &statemgmt.StateDesc{Engine: engine.EngineDirect})
			}

			assert.True(t, logdiag.HasError(ctx))
			assert.False(t, b.DeploymentBundle.StateDB.IsOpen())
			assert.Equal(t, bundledeployments.VersionCompleteVersionCompleteFailure, completion.CompletionReason)
			// Completion releases the operation buffer while retaining its final measurements.
			require.NotNil(t, b.DeploymentBundle.StateDB.OperationBufferStats())
			completed, err := b.DeploymentBundle.StateDB.CompleteVersion(ctx, true)
			require.NoError(t, err)
			assert.False(t, completed)

			// Finalize closed the WAL even though persistence failed, so ordinary recovery can retry it.
			require.NoError(t, os.Remove(statePath))
			var recovered dstate.DeploymentState
			require.NoError(t, recovered.Open(ctx, statePath, dstate.ReadingFile, dstate.WithRecovery(true), dstate.OpenDmsArgs{}))
			if operation == "unbind" {
				assert.Empty(t, recovered.GetResourceID("resources.jobs.bound"))
			} else {
				assert.Equal(t, "20", recovered.GetResourceID("resources.jobs.bound"))
			}
			_, err = recovered.Finalize(ctx)
			require.NoError(t, err)
		})
	}
}
