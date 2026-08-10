package localenv

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineErrorWrapsAndExposesCode(t *testing.T) {
	base := errors.New("boom")
	err := NewError(ErrFetch, base, "fetch %s", "x")
	assert.Equal(t, "fetch x: boom", err.Error())
	assert.Equal(t, ErrFetch, err.Code)
	assert.ErrorIs(t, err, base)
}

func TestModeString(t *testing.T) {
	assert.Equal(t, "default", ModeDefault.String())
	assert.Equal(t, "constraints-only", ModeConstraintsOnly.String())
}

func TestCommandName(t *testing.T) {
	// The --json "command" field and all help text derive from these; the
	// path must join to the full command a user types.
	assert.Equal(t, "environments setup-local", CommandName)
}

func TestNewResultEmitsEmptyArraysNotNull(t *testing.T) {
	// The --json contract requires phases/warnings to render as [] not null;
	// NewResult seeds them so consumers and golden diffs see a stable shape.
	b, err := json.Marshal(NewResult())
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, `"phases":[]`)
	assert.Contains(t, s, `"warnings":[]`)
	assert.NotContains(t, s, `"phases":null`)
	assert.NotContains(t, s, `"warnings":null`)
	// A bare Result{} literal is the shape NewResult exists to avoid.
	bare, err := json.Marshal(&Result{})
	require.NoError(t, err)
	assert.Contains(t, string(bare), `"phases":null`, "sanity: bare literal is the null case")
}

func TestComputeInfoLabel(t *testing.T) {
	cases := []struct {
		name string
		info ComputeInfo
		want string
	}{
		{"serverless", ComputeInfo{Source: "serverless", ServerlessVersion: "v4", EnvKey: "serverless/serverless-v4"}, "serverless 4"},
		{"job serverless", ComputeInfo{Source: "job", ServerlessVersion: "v5", EnvKey: "serverless/serverless-v5"}, "serverless 5"},
		{"cluster", ComputeInfo{Source: "cluster", ClusterID: "0101-abc", EnvKey: "dbr/15.4.x-scala2.12"}, "cluster 0101-abc"},
		{"fallback", ComputeInfo{Source: "bundle", EnvKey: "dbr/15.4.x-scala2.12"}, "dbr/15.4.x-scala2.12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.info.Label())
		})
	}
}
