package run

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testAppRunner struct {
	m         *mocks.MockWorkspaceClient
	b         *bundle.Bundle
	mockFiler *mockfiler.MockFiler
	ctx       context.Context
}

func (ta *testAppRunner) run(t *testing.T) {
	r := appRunner{
		key:    "my_app",
		bundle: ta.b,
		app:    ta.b.Config.Resources.Apps["my_app"],
		filerFactory: func(b *bundle.Bundle) (filer.Filer, error) {
			return ta.mockFiler, nil
		},
	}

	_, err := r.Run(ta.ctx, &Options{})
	require.NoError(t, err)
}

func setupBundle(t *testing.T) (context.Context, *bundle.Bundle, *mocks.MockWorkspaceClient) {
	root := t.TempDir()
	err := os.MkdirAll(filepath.Join(root, "my_app"), 0700)
	require.NoError(t, err)

	b := &bundle.Bundle{
		BundleRootPath: root,
		SyncRoot:       vfs.MustNew(root),
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Workspace/Users/foo@bar.com/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						App: &apps.App{
							Name: "my_app",
						},
						SourceCodePath: "./my_app",
						Config: map[string]interface{}{
							"command": []string{"echo", "hello"},
							"env": []map[string]string{
								{"name": "MY_APP", "value": "my value"},
							},
						},
					},
				},
			},
		},
	}

	mwc := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(mwc.WorkspaceClient)
	bundletest.SetLocation(b, "resources.apps.my_app", []dyn.Location{{File: "./databricks.yml"}})

	ctx := context.Background()
	ctx = cmdio.InContext(ctx, cmdio.NewIO(flags.OutputText, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "", "..."))
	ctx = cmdio.NewContext(ctx, cmdio.NewLogger(flags.ModeAppend))

	diags := bundle.Apply(ctx, b, bundle.Seq(
		mutator.DefineDefaultWorkspacePaths(),
		mutator.TranslatePaths(),
	))
	require.Empty(t, diags)

	return ctx, b, mwc
}

func setupTestApp(t *testing.T, initialAppState apps.ApplicationState, initialComputeState apps.ComputeState) *testAppRunner {
	ctx, b, mwc := setupBundle(t)

	appApi := mwc.GetMockAppsAPI()
	appApi.EXPECT().Get(mock.Anything, apps.GetAppRequest{
		Name: "my_app",
	}).Return(&apps.App{
		Name: "my_app",
		AppStatus: &apps.ApplicationStatus{
			State: initialAppState,
		},
		ComputeStatus: &apps.ComputeStatus{
			State: initialComputeState,
		},
	}, nil)

	wait := &apps.WaitGetDeploymentAppSucceeded[apps.AppDeployment]{
		Poll: func(_ time.Duration, _ func(*apps.AppDeployment)) (*apps.AppDeployment, error) {
			return nil, nil
		},
	}
	appApi.EXPECT().Deploy(mock.Anything, apps.CreateAppDeploymentRequest{
		AppName: "my_app",
		AppDeployment: &apps.AppDeployment{
			Mode:           apps.AppDeploymentModeSnapshot,
			SourceCodePath: "/Workspace/Users/foo@bar.com/files/my_app",
		},
	}).Return(wait, nil)

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(mock.Anything, "my_app/app.yml", bytes.NewBufferString(`command:
  - echo
  - hello
env:
  - name: MY_APP
    value: my value
`), filer.OverwriteIfExists).Return(nil)

	return &testAppRunner{
		m:         mwc,
		b:         b,
		mockFiler: mockFiler,
		ctx:       ctx,
	}
}

func TestAppRunStartedApp(t *testing.T) {
	r := setupTestApp(t, apps.ApplicationStateRunning, apps.ComputeStateActive)
	r.run(t)
}

func TestAppRunStoppedApp(t *testing.T) {
	r := setupTestApp(t, apps.ApplicationStateCrashed, apps.ComputeStateStopped)

	appsApi := r.m.GetMockAppsAPI()
	appsApi.EXPECT().Start(mock.Anything, apps.StartAppRequest{
		Name: "my_app",
	}).Return(&apps.WaitGetAppActive[apps.App]{
		Poll: func(_ time.Duration, _ func(*apps.App)) (*apps.App, error) {
			return &apps.App{
				Name: "my_app",
				AppStatus: &apps.ApplicationStatus{
					State: apps.ApplicationStateRunning,
				},
				ComputeStatus: &apps.ComputeStatus{
					State: apps.ComputeStateActive,
				},
				ActiveDeployment: &apps.AppDeployment{
					SourceCodePath: "/foo/bar",
					DeploymentId:   "123",
					Status: &apps.AppDeploymentStatus{
						State: apps.AppDeploymentStateInProgress,
					},
				},
				PendingDeployment: &apps.AppDeployment{
					SourceCodePath: "/foo/bar",
					DeploymentId:   "456",
					Status: &apps.AppDeploymentStatus{
						State: apps.AppDeploymentStateInProgress,
					},
				},
			}, nil
		},
	}, nil)

	appsApi.EXPECT().WaitGetDeploymentAppSucceeded(mock.Anything, "my_app", "123", mock.Anything, mock.Anything).Return(nil, nil)
	appsApi.EXPECT().WaitGetDeploymentAppSucceeded(mock.Anything, "my_app", "456", mock.Anything, mock.Anything).Return(nil, nil)
	r.run(t)
}

func TestStopApp(t *testing.T) {
	ctx, b, mwc := setupBundle(t)
	appsApi := mwc.GetMockAppsAPI()
	appsApi.EXPECT().Stop(mock.Anything, apps.StopAppRequest{
		Name: "my_app",
	}).Return(&apps.WaitGetAppStopped[apps.App]{
		Poll: func(_ time.Duration, _ func(*apps.App)) (*apps.App, error) {
			return &apps.App{
				Name: "my_app",
				AppStatus: &apps.ApplicationStatus{
					State: apps.ApplicationStateUnavailable,
				},
			}, nil
		},
	}, nil)

	r := appRunner{
		key:    "my_app",
		bundle: b,
		app:    b.Config.Resources.Apps["my_app"],
		filerFactory: func(b *bundle.Bundle) (filer.Filer, error) {
			return nil, nil
		},
	}

	err := r.Cancel(ctx)
	require.NoError(t, err)
}
