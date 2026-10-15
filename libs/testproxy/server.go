package testproxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ProxyServer struct {
	*httptest.Server

	t testutil.TestingT

	upstream         *config.Config
	httpClient       *http.Client
	RequestCallback  func(request *testserver.Request)
	ResponseCallback func(request *testserver.Request, response *testserver.EncodedResponse)
}

// New creates a reverse proxy server that sits in front of the upstream
// workspace. This is useful for recording API requests and responses in
// integration tests.
//
// Note: We cannot directly proxy the request from a localhost URL to a real
// workspace as is. This is because auth resolution in the Databricks SDK relies
// what the URL actually looks like to determine the auth method to use.
// For example, in OAuth flows, the SDK can make requests to different Microsoft
// OAuth endpoints based on the nature of the URL.
// For reference, see:
// https://github.com/databricks/databricks-sdk-go/blob/79e4b3a6e9b0b7dcb1af9ad4025deb447b01d933/common/environment/environments.go#L57
//
// An upstream with no fields set resolves a real workspace and its auth from the
// environment; an explicit host and token point at a testserver instead.
func New(t testutil.TestingT, upstream *config.Config) *ProxyServer {
	require.NoError(t, upstream.EnsureResolved())
	s := &ProxyServer{
		t:          t,
		upstream:   upstream,
		httpClient: apiclient.HTTPClient(upstream),
	}

	// Set up the proxy handler as the default handler for all requests.
	server := httptest.NewServer(http.HandlerFunc(s.proxyToUpstream))
	t.Cleanup(server.Close)

	s.Server = server
	return s
}

// includeResponseHeaders lists the response headers passed through to the
// client in addition to the status and body.
var includeResponseHeaders = []string{"X-Databricks-Org-Id"}

func (s *ProxyServer) proxyToUpstream(w http.ResponseWriter, r *http.Request) {
	request := testserver.NewRequest(s.t, r, nil)
	if s.RequestCallback != nil {
		s.RequestCallback(&request)
	}

	encodedResponse, header := s.forward(r, request.Body)

	for _, k := range includeResponseHeaders {
		if v := header.Get(k); v != "" {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(encodedResponse.StatusCode)

	_, err := w.Write(encodedResponse.Body)
	assert.NoError(s.t, err)

	if s.ResponseCallback != nil {
		s.ResponseCallback(&request, encodedResponse)
	}
}

// forward sends the request to the upstream with the upstream's credentials
// and returns its response verbatim, error bodies included, so callers see
// exactly what the workspace returned.
func (s *ProxyServer) forward(r *http.Request, body []byte) (*testserver.EncodedResponse, http.Header) {
	upstreamURL := s.upstream.CanonicalHostName() + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(s.t.Context(), r.Method, upstreamURL, reqBody)
	if err != nil {
		return internalError(err), nil
	}
	for k, v := range r.Header {
		// Authorization is set from the upstream config below. The default
		// HTTP client in Go sets Accept-Encoding to "gzip"; it is meant for
		// this proxy and will be set again for the upstream request.
		if k == "Authorization" || k == "Accept-Encoding" {
			continue
		}
		req.Header.Set(k, v[0])
	}
	if err := s.upstream.Authenticate(req); err != nil {
		return internalError(err), nil
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return internalError(err), nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return internalError(err), nil
	}
	return &testserver.EncodedResponse{StatusCode: resp.StatusCode, Body: respBody}, resp.Header
}

func internalError(err error) *testserver.EncodedResponse {
	return &testserver.EncodedResponse{
		StatusCode: 500,
		Body:       []byte(err.Error()),
	}
}
