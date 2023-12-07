package terraform

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	mock "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func mockStateFilerForPush(t *testing.T, fn func(body io.Reader)) filer.Filer {
	ctrl := gomock.NewController(t)
	mock := mock.NewMockFiler(ctrl)
	mock.
		EXPECT().
		Write(gomock.Any(), gomock.Any(), gomock.Any(), filer.CreateParentDirectories, filer.OverwriteIfExists).
		Do(func(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) error {
			fn(reader)
			return nil
		}).
		Return(nil).
		Times(1)
	return mock
}

func statePushTestBundle(t *testing.T) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
			Path: t.TempDir(),
		},
	}
}

func TestStatePush(t *testing.T) {
	mock := mockStateFilerForPush(t, func(body io.Reader) {
		dec := json.NewDecoder(body)
		var contents map[string]int
		err := dec.Decode(&contents)
		assert.NoError(t, err)
		assert.Equal(t, map[string]int{"serial": 4}, contents)
	})

	m := &statePush{
		identityFiler(mock),
	}

	ctx := context.Background()
	b := statePushTestBundle(t)

	// Write a stale local state file.
	writeLocalState(t, ctx, b, map[string]int{"serial": 4})
	err := bundle.Apply(ctx, b, m)
	assert.NoError(t, err)
}
