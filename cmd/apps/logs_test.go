package apps

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogStreamDialerConfiguresProxyAndTLS(t *testing.T) {
	t.Run("clones HTTP transport when provided", func(t *testing.T) {
		proxyURL, err := url.Parse("http://localhost:8080")
		require.NoError(t, err)

		transport := &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		}

		cfg := &config.Config{
			HTTPTransport: transport,
		}

		dialer := newLogStreamDialer(cfg)
		require.NotNil(t, dialer)

		req := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}
		actualProxy, err := dialer.Proxy(req)
		require.NoError(t, err)
		assert.Equal(t, proxyURL.String(), actualProxy.String())

		require.NotNil(t, dialer.TLSClientConfig)
		assert.NotSame(t, transport.TLSClientConfig, dialer.TLSClientConfig, "TLS config should be cloned")
		assert.Equal(t, transport.TLSClientConfig.MinVersion, dialer.TLSClientConfig.MinVersion)
	})

	t.Run("honors insecure skip verify when no transport is supplied", func(t *testing.T) {
		cfg := &config.Config{
			InsecureSkipVerify: true,
		}
		dialer := newLogStreamDialer(cfg)
		require.NotNil(t, dialer)
		require.NotNil(t, dialer.TLSClientConfig, "expected TLS config when insecure skip verify is set")
		assert.True(t, dialer.TLSClientConfig.InsecureSkipVerify)
	})
}

func TestBuildLogsURLConvertsSchemes(t *testing.T) {
	url, err := buildLogsURL("https://example.com/foo")
	require.NoError(t, err)
	assert.Equal(t, "wss://example.com/foo/logz/stream", url)

	url, err = buildLogsURL("http://example.com/foo")
	require.NoError(t, err)
	assert.Equal(t, "ws://example.com/foo/logz/stream", url)
}

func TestBuildLogsURLRejectsUnknownScheme(t *testing.T) {
	_, err := buildLogsURL("ftp://example.com/foo")
	require.Error(t, err)
}

func TestLogsMissingNameError(t *testing.T) {
	t.Chdir(t.TempDir())

	cmd := newLogsCommand()
	err := cmd.Args(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: NAME")
	assert.Contains(t, err.Error(), "Usage: logs NAME")
}

func TestNormalizeOrigin(t *testing.T) {
	assert.Equal(t, "https://example.com", normalizeOrigin("https://example.com/foo"))
	assert.Equal(t, "http://example.com", normalizeOrigin("ws://example.com/foo"))
	assert.Equal(t, "https://example.com", normalizeOrigin("wss://example.com/foo"))
	assert.Empty(t, normalizeOrigin("://invalid"))
}

func TestBuildSourceFilter(t *testing.T) {
	filters, err := buildSourceFilter([]string{"app", "system", ""})
	require.NoError(t, err)
	assert.Equal(t, map[string]struct{}{"APP": {}, "SYSTEM": {}}, filters)

	filters, err = buildSourceFilter(nil)
	require.NoError(t, err)
	assert.Nil(t, filters)

	_, err = buildSourceFilter([]string{"foo"})
	require.Error(t, err)
}

func TestWarnIfMultiInstanceAppLogs(t *testing.T) {
	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)

	warnIfMultiInstanceAppLogs(cmd, "my-app", &apps.App{ComputeMaxInstances: 2})

	assert.Contains(t, stderr.String(), `Warning: app "my-app" is configured to run up to 2 instances`)
	assert.Contains(t, stderr.String(), "streams logs from a single app instance")
}

func TestWarnIfMultiInstanceAppLogsSkipsSingleInstanceApps(t *testing.T) {
	for _, app := range []*apps.App{
		{},
		{ComputeMinInstances: 1, ComputeMaxInstances: 1},
	} {
		cmd := &cobra.Command{}
		stderr := &bytes.Buffer{}
		cmd.SetErr(stderr)

		warnIfMultiInstanceAppLogs(cmd, "my-app", app)

		assert.Empty(t, stderr.String())
	}
}

func TestLogStreamRoutingCookie(t *testing.T) {
	cookie := logStreamRoutingCookie(&apps.App{ComputeMinInstances: 1, ComputeMaxInstances: 3})

	value := strings.TrimPrefix(cookie, appRouterCookieName+"=")
	require.NotEqual(t, cookie, value)
	_, err := uuid.Parse(value)
	require.NoError(t, err)
}

func TestLogStreamRoutingCookieSkipsSingleInstanceApps(t *testing.T) {
	for _, app := range []*apps.App{
		{},
		{ComputeMinInstances: 1, ComputeMaxInstances: 1},
	} {
		assert.Empty(t, logStreamRoutingCookie(app))
	}
}
