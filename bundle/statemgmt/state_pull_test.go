package statemgmt

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
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func mockStateFilerForPull(t *testing.T, contents map[string]any, merr error) filer.Filer {
	buf, err := json.Marshal(contents)
	assert.NoError(t, err)

	f := mockfiler.NewMockFiler(t)
	f.
		EXPECT().
		Read(mock.Anything, mock.Anything).
		Return(io.NopCloser(bytes.NewReader(buf)), merr).
		Maybe()
	return f
}

func statePullTestBundle(t *testing.T) *bundle.Bundle {
	return &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
		},
	}
}

func TestStatePullLocalErrorWhenRemoteHasNoLineage(t *testing.T) {
	m := &statePull{}

	t.Run("no local state", func(t *testing.T) {
		m.filerFactory = identityFiler(mockStateFilerForPull(t, map[string]any{"serial": 5}, nil))

		ctx := context.Background()
		b := statePullTestBundle(t)
		diags := bundle.Apply(ctx, b, m)
		assert.EqualError(t, diags.Error(), "remote state file does not have a lineage")
	})

	t.Run("local state with lineage", func(t *testing.T) {
		m.filerFactory = identityFiler(mockStateFilerForPull(t, map[string]any{"serial": 5}, nil))

		ctx := context.Background()
		b := statePullTestBundle(t)
		writeLocalState(t, ctx, b, map[string]any{"serial": 5, "lineage": "aaaa"})

		diags := bundle.Apply(ctx, b, m)
		assert.EqualError(t, diags.Error(), "remote state file does not have a lineage")
	})
}

func TestStatePullLocal(t *testing.T) {
	tcases := []struct {
		name     string
		remote   map[string]any
		local    map[string]any
		expected map[string]any
	}{
		{
			name:     "remote missing, local missing",
			remote:   nil,
			local:    nil,
			expected: nil,
		},
		{
			name:     "remote missing, local present",
			remote:   nil,
			local:    map[string]any{"serial": 5, "lineage": "aaaa"},
			expected: map[string]any{"serial": float64(5), "lineage": "aaaa"},
		},
		{
			name:     "local stale",
			remote:   map[string]any{"serial": 10, "lineage": "aaaa", "some_other_key": 123},
			local:    map[string]any{"serial": 5, "lineage": "aaaa"},
			expected: map[string]any{"serial": float64(10), "lineage": "aaaa", "some_other_key": float64(123)},
		},
		{
			name:     "local equal",
			remote:   map[string]any{"serial": 5, "lineage": "aaaa", "some_other_key": 123},
			local:    map[string]any{"serial": 5, "lineage": "aaaa"},
			expected: map[string]any{"serial": float64(5), "lineage": "aaaa"},
		},
		{
			name:     "local newer",
			remote:   map[string]any{"serial": 5, "lineage": "aaaa", "some_other_key": 123},
			local:    map[string]any{"serial": 6, "lineage": "aaaa"},
			expected: map[string]any{"serial": float64(6), "lineage": "aaaa"},
		},
		{
			name:     "remote and local have different lineages",
			remote:   map[string]any{"serial": 5, "lineage": "aaaa"},
			local:    map[string]any{"serial": 10, "lineage": "bbbb"},
			expected: map[string]any{"serial": float64(5), "lineage": "aaaa"},
		},
		{
			name:     "local is missing lineage",
			remote:   map[string]any{"serial": 5, "lineage": "aaaa"},
			local:    map[string]any{"serial": 10},
			expected: map[string]any{"serial": float64(5), "lineage": "aaaa"},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			m := &statePull{}
			if tc.remote == nil {
				m.filerFactory = identityFiler(mockStateFilerForPull(t, nil, os.ErrNotExist))
			} else {
				m.filerFactory = identityFiler(mockStateFilerForPull(t, tc.remote, nil))
			}

			ctx := context.Background()
			b := statePullTestBundle(t)
			if tc.local != nil {
				writeLocalState(t, ctx, b, tc.local)
			}

			diags := bundle.Apply(ctx, b, m)
			assert.NoError(t, diags.Error())

			if tc.expected == nil {
				_, err := os.Stat(localStateFile(t, ctx, b))
				assert.ErrorIs(t, err, fs.ErrNotExist)
			} else {
				localState := readLocalState(t, ctx, b)
				assert.Equal(t, tc.expected, localState)
			}
		})
	}
}
