package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupPrimaryProvisionedURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"url": "https://dbc-abc123.cloud.databricks.com"}`))
	}))
	defer server.Close()

	url, err := LookupPrimaryProvisionedURL(t.Context(), server.URL, "abc-123", "test-token", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://dbc-abc123.cloud.databricks.com", url)
}

func TestLookupPrimaryProvisionedURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := LookupPrimaryProvisionedURL(t.Context(), server.URL, "abc-123", "test-token", nil)
	assert.ErrorContains(t, err, "status 404")
}

func TestLookupPrimaryProvisionedURL_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	_, err := LookupPrimaryProvisionedURL(t.Context(), server.URL, "abc-123", "test-token", nil)
	assert.ErrorContains(t, err, "decoding provisioned-urls response")
}

func TestLookupPrimaryProvisionedURL_VerifyRequestDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/accounts/abc-123/provisioned-urls/primary", r.URL.Path)
		assert.Equal(t, "Bearer my-secret-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"url": "https://dbc-abc123.cloud.databricks.com"}`))
	}))
	defer server.Close()

	_, err := LookupPrimaryProvisionedURL(t.Context(), server.URL, "abc-123", "my-secret-token", nil)
	require.NoError(t, err)
}
