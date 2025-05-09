package apps

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAppUploadConfig(t *testing.T) {
	root := t.TempDir()
	err := os.MkdirAll(filepath.Join(root, "my_app"), 0o700)
	require.NoError(t, err)

	b := &bundle.Bundle{
		BundleRootPath: root,
		SyncRootPath:   root,
		SyncRoot:       vfs.MustNew(root),
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Workspace/Users/foo@bar.com/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						App: apps.App{
							Name: "my_app",
						},
						SourceCodePath: "./my_app",
						Config: map[string]any{
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

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(mock.Anything, "my_app/app.yml", bytes.NewBufferString(`command:
  - echo
  - hello
env:
  - name: MY_APP
    value: my value
`), filer.OverwriteIfExists).Return(nil)

	u := uploadConfig{
		filerFactory: func(b *bundle.Bundle) (filer.Filer, error) {
			return mockFiler, nil
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(root, "databricks.yml")}})

	diags := bundle.ApplySeq(context.Background(), b, mutator.TranslatePaths(), &u)
	require.NoError(t, diags.Error())
}
