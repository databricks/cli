package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	mock "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func mockStateFilerForPull(t *testing.T, contents map[string]int, merr error) filer.Filer {
	buf, err := json.Marshal(contents)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	filer := mock.NewMockFiler(ctrl)
	filer.
		EXPECT().
		Read(gomock.Any(), gomock.Eq(TerraformStateFileName)).
		Return(io.NopCloser(bytes.NewReader(buf)), merr).
		Times(1)
	return filer
}

func statePullTestBundle(t *testing.T) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
			Path: t.TempDir(),
		},
	}
}

func TestStatePullLocalMissingRemoteMissing(t *testing.T) {
	m := &statePull{
		identityFiler(mockStateFilerForPull(t, nil, os.ErrNotExist)),
	}

	ctx := context.Background()
	b := statePullTestBundle(t)
	err := bundle.Apply(ctx, b, m)
	assert.NoError(t, err)

	// Confirm that no local state file has been written.
	_, err = os.Stat(localStateFile(t, ctx, b))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestStatePullLocalMissingRemotePresent(t *testing.T) {
	m := &statePull{
		identityFiler(mockStateFilerForPull(t, map[string]int{"serial": 5}, nil)),
	}

	ctx := context.Background()
	b := statePullTestBundle(t)
	err := bundle.Apply(ctx, b, m)
	assert.NoError(t, err)

	// Confirm that the local state file has been updated.
	localState := readLocalState(t, ctx, b)
	assert.Equal(t, map[string]int{"serial": 5}, localState)
}

func TestStatePullLocalStale(t *testing.T) {
	m := &statePull{
		identityFiler(mockStateFilerForPull(t, map[string]int{"serial": 5}, nil)),
	}

	ctx := context.Background()
	b := statePullTestBundle(t)

	// Write a stale local state file.
	writeLocalState(t, ctx, b, map[string]int{"serial": 4})
	err := bundle.Apply(ctx, b, m)
	assert.NoError(t, err)

	// Confirm that the local state file has been updated.
	localState := readLocalState(t, ctx, b)
	assert.Equal(t, map[string]int{"serial": 5}, localState)
}

func TestStatePullLocalEqual(t *testing.T) {
	m := &statePull{
		identityFiler(mockStateFilerForPull(t, map[string]int{"serial": 5, "some_other_key": 123}, nil)),
	}

	ctx := context.Background()
	b := statePullTestBundle(t)

	// Write a local state file with the same serial as the remote.
	writeLocalState(t, ctx, b, map[string]int{"serial": 5})
	err := bundle.Apply(ctx, b, m)
	assert.NoError(t, err)

	// Confirm that the local state file has not been updated.
	localState := readLocalState(t, ctx, b)
	assert.Equal(t, map[string]int{"serial": 5}, localState)
}

func TestStatePullLocalNewer(t *testing.T) {
	m := &statePull{
		identityFiler(mockStateFilerForPull(t, map[string]int{"serial": 5, "some_other_key": 123}, nil)),
	}

	ctx := context.Background()
	b := statePullTestBundle(t)

	// Write a local state file with a newer serial as the remote.
	writeLocalState(t, ctx, b, map[string]int{"serial": 6})
	err := bundle.Apply(ctx, b, m)
	assert.NoError(t, err)

	// Confirm that the local state file has not been updated.
	localState := readLocalState(t, ctx, b)
	assert.Equal(t, map[string]int{"serial": 6}, localState)
}
