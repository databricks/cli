package internal_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/acceptance/internal"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dmsPath = "/api/2.0/bundle/deployments"

// Generate a short-lived fixture instead of distributing LITE's private key.
func liteboxCertificate(t *testing.T) (tls.Certificate, string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"deployment-metadata-service.svc.cluster.local"},
		NotBefore:    time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:        true, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	certFile, keyFile := filepath.Join(t.TempDir(), "cert.pem"), filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(certFile, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0o600))
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	return pair, certFile, keyFile
}

func TestDMSLiteboxRoutesAndRecords(t *testing.T) {
	cert, certFile, keyFile := liteboxCertificate(t)
	roots := x509.NewCertPool()
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	roots.AddCert(parsed)
	var calls atomic.Int32
	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "page_token=a%2Fb", r.URL.RawQuery)
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Equal(t, "456", r.Header.Get("X-Databricks-Org-Id"))
		assert.Equal(t, "123", r.Header.Get("X-Databricks-User-Id"))
		assert.Equal(t, "alice@databricks.com", r.Header.Get("X-Databricks-User-Name"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Len(t, r.TLS.PeerCertificates, 1)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"id":9007199254740993}`, string(body))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err = w.Write(body)
		assert.NoError(t, err)
	}))
	upstream.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: roots}
	upstream.StartTLS()
	t.Cleanup(upstream.Close)
	t.Setenv("DMS_LITEBOX_URL", upstream.URL)
	t.Setenv("DMS_LITEBOX_CERT", certFile)
	t.Setenv("DMS_LITEBOX_CA", certFile)
	t.Setenv("DMS_LITEBOX_KEY", keyFile)
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	var recorded atomic.Int32
	server.ResponseCallback = func(_ *testserver.Request, response *testserver.EncodedResponse) {
		if response.StatusCode == http.StatusServiceUnavailable {
			recorded.Add(1)
		}
	}
	internal.ConfigureDMSLitebox(t, server)
	// Cover a registered fake route and an unknown DMS route (no mock fallback).
	for _, path := range []string{dmsPath, dmsPath + "/unknown/route"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+path+"?page_token=a%2Fb", strings.NewReader(`{"id":9007199254740993}`))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer fake-token")
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Equal(t, "2", resp.Header.Get("Retry-After"))
		assert.JSONEq(t, `{"id":9007199254740993}`, string(body))
	}
	resp, err := server.Client().Get(server.URL + "/.well-known/databricks-config")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 2, calls.Load())
	assert.EqualValues(t, 2, recorded.Load())
}

func TestDMSLiteboxFailsClosed(t *testing.T) {
	for _, failure := range []string{"untrusted certificate", "unreachable server"} {
		t.Run(failure, func(t *testing.T) {
			_, certFile, keyFile := liteboxCertificate(t)
			// The server's unrelated certificate must not be trusted by the adapter.
			upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("request reached an untrusted upstream")
			}))
			t.Cleanup(upstream.Close)
			if failure == "unreachable server" {
				upstream.Close()
			}
			t.Setenv("DMS_LITEBOX_URL", upstream.URL)
			t.Setenv("DMS_LITEBOX_CERT", certFile)
			t.Setenv("DMS_LITEBOX_CA", certFile)
			t.Setenv("DMS_LITEBOX_KEY", keyFile)
			server := testserver.New(t)
			testserver.AddDefaultHandlers(server)
			server.Handle(http.MethodGet, dmsPath, func(testserver.Request) any { return "mock" })
			internal.ConfigureDMSLitebox(t, server)
			resp, err := server.Client().Get(server.URL + dmsPath)
			require.NoError(t, err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
			assert.Contains(t, string(body), "LITEBOX_CONNECTION_ERROR")
		})
	}
}

func TestDMSLiteboxDisabled(t *testing.T) {
	t.Setenv("DMS_LITEBOX_URL", "")
	server := testserver.New(t)
	server.Handle("GET", dmsPath, func(testserver.Request) any { return "mock" })
	internal.ConfigureDMSLitebox(t, server)
	resp, err := server.Client().Get(server.URL + dmsPath)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "mock", string(body))
}
