package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func mockStateFilerForPush(t *testing.T, fn func(body io.Reader)) filer.Filer {
	f := mockfiler.NewMockFiler(t)
	f.
		EXPECT().
		Write(mock.Anything, mock.Anything, mock.Anything, filer.CreateParentDirectories, filer.OverwriteIfExists).
		Run(func(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) {
			fn(reader)
		}).
		Return(nil).
		Times(1)
	return f
}

func statePushTestBundle(t *testing.T) *bundle.Bundle {
	return &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
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
	writeLocalState(t, ctx, b, map[string]any{"serial": 4})
	diags := bundle.Apply(ctx, b, m)
	assert.NoError(t, diags.Error())
}

func TestStatePushLargeState(t *testing.T) {
	mock := mockfiler.NewMockFiler(t)
	m := &statePush{
		identityFiler(mock),
	}

	ctx := context.Background()
	b := statePushTestBundle(t)

	largeState := map[string]any{}
	for i := range 1000000 {
		largeState[fmt.Sprintf("field_%d", i)] = i
	}

	// Write a stale local state file.
	writeLocalState(t, ctx, b, largeState)
	diags := bundle.Apply(ctx, b, m)
	assert.ErrorContains(t, diags.Error(), "Terraform state file size exceeds the maximum allowed size of 10485760 bytes. Please reduce the number of resources in your bundle, split your bundle into multiple or re-run the command with --force flag")

	// Force the write.
	b = statePushTestBundle(t)
	b.Config.Bundle.Force = true
	diags = bundle.Apply(ctx, b, m)
	assert.NoError(t, diags.Error())
}
