package apps

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogStreamDialerClonesHTTPTransport(t *testing.T) {
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
	require.NotNil(t, dialer.Proxy)

	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}
	actualProxy, err := dialer.Proxy(req)
	require.NoError(t, err)
	assert.Equal(t, proxyURL.String(), actualProxy.String())

	require.NotNil(t, dialer.TLSClientConfig)
	assert.NotSame(t, transport.TLSClientConfig, dialer.TLSClientConfig, "TLS config should be cloned")
	assert.Equal(t, transport.TLSClientConfig.MinVersion, dialer.TLSClientConfig.MinVersion)
}

func TestNewLogStreamDialerHonorsInsecureSkipVerify(t *testing.T) {
	cfg := &config.Config{
		InsecureSkipVerify: true,
	}

	dialer := newLogStreamDialer(cfg)
	require.NotNil(t, dialer)
	require.NotNil(t, dialer.TLSClientConfig, "expected TLS config when insecure skip verify is set")
	assert.True(t, dialer.TLSClientConfig.InsecureSkipVerify)
}
