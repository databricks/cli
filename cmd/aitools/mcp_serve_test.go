package aitools

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/databricks-sdk-go/config"
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
	assert.Contains(t, err.Error(), "databricks auth login --profile logfood")
	assert.Equal(t, context.Canceled, ctx.Err())

	// A second failure keeps the first error for the exit status.
	_ = g.authenticateRequest(&http.Request{Header: http.Header{}})
	assert.Equal(t, 2, calls)
	assert.ErrorIs(t, g.failure(), cause)
}

// The refresh failure reaches the user as the AUTH_FAILED error, which names the profile and its login command
// and exits 2.
func TestRefreshGuardStoresAuthError(t *testing.T) {
	cause := errors.New("refresh token expired")
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return cause },
		profile:      "my profile",
		cancel:       func() {},
	}

	_ = g.authenticateRequest(&http.Request{Header: http.Header{}})

	be, ok := errors.AsType[*mcp.BindError](g.failure())
	require.True(t, ok)
	assert.Equal(t, mcp.BindAuthFailed, be.Category)
	assert.ErrorIs(t, be, cause)
	assert.Contains(t, be.Error(), "databricks auth login --profile 'my profile'")
	assert.Equal(t, 2, root.ExitStatus(g.failure()))
}

func TestServeFailureSetsExitStatusWithoutReprinting(t *testing.T) {
	cmd := NewMcpServeCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cause := errors.New("MCP server x is bound to Databricks profile")

	err := serveFailure(cmd, cause)

	assert.Equal(t, "MCP server x is bound to Databricks profile\n", stderr.String())
	assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 1, root.ExitStatus(err))

	bindErr := mcp.AuthError("logfood", cause)
	stderr.Reset()
	err = serveFailure(cmd, bindErr)

	assert.Equal(t, bindErr.Error()+"\n", stderr.String())
	assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
}

func TestServeTransportHasConnectTimeout(t *testing.T) {
	assert.Equal(t, 10*time.Second, serveDialer().Timeout)

	tr := serveTransport(&config.Config{})
	assert.NotNil(t, tr.DialContext)
	require.NotNil(t, tr.TLSClientConfig)
	assert.False(t, tr.TLSClientConfig.InsecureSkipVerify)
}

func TestServeTransportSkipsTLSVerifyWhenProfileSetsIt(t *testing.T) {
	tr := serveTransport(&config.Config{InsecureSkipVerify: true})
	require.NotNil(t, tr.TLSClientConfig)
	assert.True(t, tr.TLSClientConfig.InsecureSkipVerify)
}

func TestServeTransportKeepsProfileHTTPTransportTLS(t *testing.T) {
	base := &http.Transport{TLSClientConfig: &tls.Config{ServerName: "profile.test"}}
	tr := serveTransport(&config.Config{HTTPTransport: base})
	require.NotNil(t, tr.TLSClientConfig)
	assert.Equal(t, "profile.test", tr.TLSClientConfig.ServerName)
	assert.NotSame(t, base.TLSClientConfig, tr.TLSClientConfig)
}
