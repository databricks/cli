package run

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeApps serves the apps API calls made by appRunner and records them.
type fakeApps struct {
	t *testing.T

	// app is returned by GET /api/2.0/apps/my_app.
	app map[string]any
	// startedApp replaces app after a start request.
	startedApp map[string]any

	// deployErrors is the number of deploy requests to reject before accepting one.
	deployErrors int

	starts      int
	stops       int
	deployments []map[string]any
	waited      []string
}

func newFakeApps(t *testing.T, server *testserver.Server) *fakeApps {
	f := &fakeApps{t: t}
	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		assert.Equal(t, "my_app", req.Vars["name"])
		return f.app
	})
	server.Handle("POST", "/api/2.0/apps/{name}/start", func(req testserver.Request) any {
		f.starts++
		f.app = f.startedApp
		return f.app
	})
	server.Handle("POST", "/api/2.0/apps/{name}/stop", func(req testserver.Request) any {
		f.stops++
		f.app = appWithState("UNAVAILABLE", "STOPPED")
		return f.app
	})
	server.Handle("POST", "/api/2.0/apps/{name}/deployments", func(req testserver.Request) any {
		if f.deployErrors > 0 {
			f.deployErrors--
			return testserver.Response{
				StatusCode: 400,
				Body:       map[string]string{"error_code": "INVALID_STATE", "message": "deployment in progress"},
			}
		}
		var body map[string]any
		assert.NoError(t, json.Unmarshal(req.Body, &body))
		f.deployments = append(f.deployments, body)
		return map[string]any{"deployment_id": "new_deployment_id", "status": map[string]any{"state": "IN_PROGRESS"}}
	})
	server.Handle("GET", "/api/2.0/apps/{name}/deployments/{id}", func(req testserver.Request) any {
		f.waited = append(f.waited, req.Vars["id"])
		return map[string]any{"deployment_id": req.Vars["id"], "status": map[string]any{"state": "SUCCEEDED"}}
	})
	return f
}

func (f *fakeApps) run(ctx context.Context, b *bundle.Bundle) {
	r := appRunner{
		key:    "my_app",
		bundle: b,
		app:    b.Config.Resources.Apps["my_app"],
	}

	_, err := r.Run(ctx, &Options{})
	require.NoError(f.t, err)
}

func (f *fakeApps) assertDeployedOnce() {
	expected := []map[string]any{{
		"mode":             "SNAPSHOT",
		"source_code_path": "/Workspace/Users/foo@bar.test/files/my_app",
	}}
	assert.Equal(f.t, expected, f.deployments)
}

func setupBundle(t *testing.T) (context.Context, *bundle.Bundle, *testserver.Server) {
	root := t.TempDir()
	err := os.MkdirAll(filepath.Join(root, "my_app"), 0o700)
	require.NoError(t, err)

	b := &bundle.Bundle{
		BundleRootPath: root,
		SyncRootPath:   root,
		SyncRoot:       vfs.MustNew(root),
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Workspace/Users/foo@bar.test/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						App: apps.App{
							Name:           new("my_app"),
							SourceCodePath: new("./my_app"),
						},
					},
				},
			},
		},
	}

	w, server := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)
	bundletest.SetLocation(b, "resources.apps.my_app", []diag.Location{{File: filepath.Join(root, "./databricks.yml")}})

	ctx := cmdio.MockDiscard(t.Context())

	diags := bundle.ApplySeq(ctx, b,
		mutator.DefineDefaultWorkspacePaths(),
		mutator.NormalizePaths(),
		mutator.TranslatePaths(),
	)
	require.Empty(t, diags)

	return ctx, b, server
}

func appWithState(appState, computeState string) map[string]any {
	return map[string]any{
		"name":           "my_app",
		"app_status":     map[string]any{"state": appState},
		"compute_status": map[string]any{"state": computeState},
	}
}

func TestAppRunStartedApp(t *testing.T) {
	ctx, b, server := setupBundle(t)
	f := newFakeApps(t, server)
	f.app = appWithState("RUNNING", "ACTIVE")

	f.run(ctx, b)
	assert.Equal(t, 0, f.starts)
	f.assertDeployedOnce()
}

func TestAppRunStoppedApp(t *testing.T) {
	ctx, b, server := setupBundle(t)
	f := newFakeApps(t, server)
	f.app = appWithState("CRASHED", "STOPPED")
	f.startedApp = appWithState("RUNNING", "ACTIVE")

	f.run(ctx, b)
	assert.Equal(t, 1, f.starts)
	f.assertDeployedOnce()
}

func TestAppRunWithAnActiveDeploymentInProgress(t *testing.T) {
	ctx, b, server := setupBundle(t)
	f := newFakeApps(t, server)
	f.app = appWithState("CRASHED", "STOPPED")
	f.startedApp = appWithState("RUNNING", "ACTIVE")
	f.startedApp["active_deployment"] = map[string]any{
		"deployment_id": "active_deployment_id",
		"status":        map[string]any{"state": "IN_PROGRESS"},
	}
	f.startedApp["pending_deployment"] = map[string]any{
		"deployment_id": "pending_deployment_id",
		"status":        map[string]any{"state": "CANCELLED"},
	}

	f.run(ctx, b)
	assert.Equal(t, []string{"active_deployment_id", "new_deployment_id"}, f.waited)
	f.assertDeployedOnce()
}

func TestAppDeployWithDeploymentInProgress(t *testing.T) {
	ctx, b, server := setupBundle(t)
	f := newFakeApps(t, server)
	f.app = appWithState("RUNNING", "ACTIVE")
	f.app["active_deployment"] = map[string]any{
		"deployment_id": "active_deployment_id",
		"status":        map[string]any{"state": "IN_PROGRESS"},
	}
	// The first deployment is rejected; the runner waits for the active
	// deployment and retries.
	f.deployErrors = 1

	f.run(ctx, b)
	assert.Equal(t, []string{"active_deployment_id", "new_deployment_id"}, f.waited)
	f.assertDeployedOnce()
}

func TestStopApp(t *testing.T) {
	ctx, b, server := setupBundle(t)
	f := newFakeApps(t, server)

	r := appRunner{
		key:    "my_app",
		bundle: b,
		app:    b.Config.Resources.Apps["my_app"],
	}

	err := r.Cancel(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, f.stops)
}

func TestBuildAppDeploymentWithValueFrom(t *testing.T) {
	app := &resources.App{
		SourceCodePath: new("/path/to/app"),
		Config: &resources.AppConfig{
			Command: []string{"python", "app.py"},
			Env: []resources.AppEnvVar{
				{
					Name:  "REGULAR_VAR",
					Value: "regular_value",
				},
				{
					Name:      "SECRET_VAR",
					ValueFrom: "secrets/my-secret",
				},
			},
		},
	}

	sdkDeployment, err := appdeploy.BuildDeployment(utils.Deref(app.SourceCodePath), app.Config, app.GitSource)
	require.NoError(t, err)
	deployment, err := apps.AppDeploymentFromSDK(sdkDeployment)
	require.NoError(t, err)

	require.Equal(t, apps.AppDeployment_Mode_Snapshot, deployment.Mode)
	require.Equal(t, "/path/to/app", utils.Deref(deployment.SourceCodePath))
	require.Equal(t, []string{"python", "app.py"}, deployment.Command)

	require.Equal(t, []apps.EnvVar{
		{Name: new("REGULAR_VAR"), Value: new("regular_value")},
		{Name: new("SECRET_VAR"), ValueFrom: new("secrets/my-secret")},
	}, deployment.EnvVars)
}

// value and value_from are a oneof in the Apps API.
func TestBuildAppDeploymentRejectsValueAndValueFrom(t *testing.T) {
	config := &resources.AppConfig{
		Env: []resources.AppEnvVar{{
			Name:      "COMBINED_VAR",
			Value:     "default_value",
			ValueFrom: "secrets/override",
		}},
	}

	_, err := appdeploy.BuildDeployment("/path/to/app", config, nil)
	require.ErrorContains(t, err, "only one of value, value_from can be set")
}
