package aitools

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpServeIsHidden(t *testing.T) {
	assert.True(t, NewMcpServeCmd().Hidden)
}

func TestRefreshGuardPassesThroughWhileTokensRefresh(t *testing.T) {
	cancelled := false
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return nil },
		profile:      "logfood",
		cancel:       func() { cancelled = true },
	}

	require.NoError(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))
	assert.NoError(t, g.failure())
	assert.False(t, cancelled)
}

func TestRefreshGuardStopsProxyOnFirstFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	cause := errors.New("refresh token expired")
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			calls++
			return cause
		},
		profile: "logfood",
		cancel:  cancel,
	}

	err := g.authenticateRequest(&http.Request{Header: http.Header{}})
	require.Error(t, err)
	assert.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), `profile logfood`)
	assert.Contains(t, err.Error(), "databricks auth login --profile logfood")
	assert.Equal(t, context.Canceled, ctx.Err())

	// A second failure keeps the first error for the exit status.
	_ = g.authenticateRequest(&http.Request{Header: http.Header{}})
	assert.Equal(t, 2, calls)
	assert.ErrorIs(t, g.failure(), cause)
}

func TestServeFailureSetsExitStatusWithoutReprinting(t *testing.T) {
	cmd := NewMcpServeCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cause := errors.New("MCP server x is bound to Databricks profile")

	err := serveFailure(cmd, serveExitCode, cause)

	assert.Equal(t, "MCP server x is bound to Databricks profile\n", stderr.String())
	assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
}
