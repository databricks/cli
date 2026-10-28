package testproxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/fault"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const faultEndpoint = "/__testserver/fault"

type ProxyServer struct {
	*httptest.Server

	t                testutil.TestingT
	upstream         *url.URL
	transport        http.RoundTripper
	token            string
	faults           *fault.Rules
	RequestCallback  func(request *testserver.Request)
	ResponseCallback func(request *testserver.Request, response *testserver.EncodedResponse)
}

// New creates a reverse proxy with token-scoped fault injection and optional recording.
// The upstream config resolves real workspace credentials; token identifies the test
// locally and is never forwarded. Resolving auth against localhost would break OAuth:
// https://github.com/databricks/databricks-sdk-go/blob/79e4b3a6e9b0b7dcb1af9ad4025deb447b01d933/common/environment/environments.go#L57
func New(t testutil.TestingT, upstream *config.Config, token string) *ProxyServer {
	require.NoError(t, upstream.EnsureResolved())
	upstreamURL, err := url.Parse(upstream.Host)
	require.NoError(t, err)

	transport := upstream.HTTPTransport
	if transport == nil {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: upstream.InsecureSkipVerify} //nolint:gosec // Honor the upstream SDK config.
		t.Cleanup(tr.CloseIdleConnections)
		transport = tr
	}
	s := &ProxyServer{
		t:         t,
		upstream:  upstreamURL,
		transport: &authenticatedTransport{upstream: upstream, next: transport},
		token:     token,
		faults:    fault.NewRules(),
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.proxyToUpstream))
	t.Cleanup(s.Close)
	return s
}

type authenticatedTransport struct {
	upstream *config.Config
	next     http.RoundTripper
}

func (tr *authenticatedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Del("Authorization")
	if err := tr.upstream.Authenticate(r); err != nil {
		return nil, fmt.Errorf("authenticate upstream request: %w", err)
	}
	for key, value := range auth.WorkspaceIDHeaders(tr.upstream) {
		r.Header.Set(key, value)
	}
	if tr.upstream.Headers != nil {
		if err := tr.upstream.Headers(r); err != nil {
			return nil, err
		}
	}
	return tr.next.RoundTrip(r)
}

func (s *ProxyServer) proxyToUpstream(w http.ResponseWriter, r *http.Request) {
	request := testserver.NewRequest(s.t, r, nil)
	if strings.HasPrefix(r.URL.Path, "/__testserver/") {
		if r.URL.Path != faultEndpoint || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if request.Token == "" || request.Token != s.token {
			http.Error(w, "invalid test token", http.StatusUnauthorized)
			return
		}
		if err := s.faults.Register(request.Token, request.Body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.RequestCallback != nil {
		s.RequestCallback(&request)
	}
	rule := s.faults.Check(r.Method, r.URL.Path, request.Token, request.Body)
	if rule != nil && !rule.AfterHandler {
		s.writeResponse(w, &request, injectedResponse(rule))
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(request.Body))
	proxy := httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(s.upstream)
			// Let the transport negotiate compression so response logging sees decoded bytes.
			pr.Out.Header.Del("Accept-Encoding")
			// Preserve the client's query encoding and repeated parameters.
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
		},
		Transport: s.transport,
		ModifyResponse: func(response *http.Response) error {
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				return err
			}
			encoded := testserver.EncodedResponse{
				StatusCode: response.StatusCode,
				Headers:    response.Header,
				Body:       body,
			}
			if rule != nil {
				encoded = injectedResponse(rule)
			}
			if s.ResponseCallback != nil {
				s.ResponseCallback(&request, &encoded)
			}
			response.StatusCode = encoded.StatusCode
			response.Header = encoded.Headers
			response.Body = io.NopCloser(bytes.NewReader(encoded.Body))
			response.ContentLength = int64(len(encoded.Body))
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			s.writeResponse(w, &request, testserver.EncodedResponse{
				StatusCode: http.StatusBadGateway,
				Body:       []byte(err.Error()),
			})
		},
	}
	proxy.ServeHTTP(w, r)
}

func injectedResponse(rule *fault.Rule) testserver.EncodedResponse {
	return testserver.EncodedResponse{
		StatusCode: rule.StatusCode,
		Headers:    http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(rule.Body),
	}
}

func (s *ProxyServer) writeResponse(w http.ResponseWriter, request *testserver.Request, response testserver.EncodedResponse) {
	if s.ResponseCallback != nil {
		s.ResponseCallback(request, &response)
	}
	maps.Copy(w.Header(), response.Headers)
	w.WriteHeader(response.StatusCode)
	_, err := w.Write(response.Body)
	assert.NoError(s.t, err)
}
