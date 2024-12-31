package deploy

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStatePush(t *testing.T) {
	s := &statePush{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		f.EXPECT().Write(mock.Anything, DeploymentStateFileName, mock.MatchedBy(func(r *os.File) bool {
			bytes, err := io.ReadAll(r)
			if err != nil {
				return false
			}

			var state DeploymentState
			err = json.Unmarshal(bytes, &state)
			if err != nil {
				return false
			}

			if state.Seq != 1 {
				return false
			}

			if len(state.Files) != 1 {
				return false
			}

			return true
		}), filer.CreateParentDirectories, filer.OverwriteIfExists).Return(nil)
		return f, nil
	}}

	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
			},
		},
	}

	ctx := context.Background()

	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	state := DeploymentState{
		Version: DeploymentStateVersion,
		Seq:     1,
		Files: []File{
			{
				LocalPath: "bar/t1.py",
			},
		},
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	err = os.WriteFile(statePath, data, 0o644)
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, s)
	require.NoError(t, diags.Error())
}
