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
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := LookupPrimaryProvisionedURL(t.Context(), server.URL, "abc-123", "test-token", nil)
	assert.ErrorContains(t, err, "status 500")
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

func TestLookupPrimaryProvisionedURL_EscapesAccountID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/accounts/a%2Fb%3Fc/provisioned-urls/primary", r.URL.EscapedPath())
		assert.Empty(t, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"url": "https://acme.databricks.com"}`))
	}))
	defer server.Close()

	_, err := LookupPrimaryProvisionedURL(t.Context(), server.URL, "a/b?c", "token", nil)
	require.NoError(t, err)
}

func TestLookupPrimaryProvisionedURL_NotFoundMeansNoPrimaryURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"Primary provisioned URL does not exist"}`))
	}))
	defer server.Close()

	url, err := LookupPrimaryProvisionedURL(t.Context(), server.URL, "abc", "token", nil)
	require.NoError(t, err)
	assert.Empty(t, url)
}

func TestLookupWorkspacePrimaryURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.well-known/databricks-config", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("include_primary_url"))
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"workspace_id": "123", "primary_url": "https://acme.databricks.com"}`))
	}))
	defer server.Close()

	resp, err := LookupWorkspacePrimaryURL(t.Context(), server.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "https://acme.databricks.com", resp.PrimaryURL)
	assert.Equal(t, "123", resp.WorkspaceID)
}

func TestLookupWorkspacePrimaryURL_NoPrimaryURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"workspace_id": "123"}`))
	}))
	defer server.Close()

	resp, err := LookupWorkspacePrimaryURL(t.Context(), server.URL, nil)
	require.NoError(t, err)
	assert.Empty(t, resp.PrimaryURL)
}

func TestLookupWorkspacePrimaryURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := LookupWorkspacePrimaryURL(t.Context(), server.URL, nil)
	assert.ErrorContains(t, err, "databricks-config endpoint returned status 404")
}

func TestLookupOAuthTokenEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/oidc/.well-known/oauth-authorization-server", r.URL.Path)
		assert.Equal(t, "123", r.URL.Query().Get("o"))
		_, _ = w.Write([]byte(`{"token_endpoint": "https://dbc-123.cloud.databricks.test/oidc/v1/token"}`))
	}))
	defer server.Close()

	endpoint, err := LookupOAuthTokenEndpoint(t.Context(), server.URL+"/oidc/.well-known/oauth-authorization-server?o=123", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://dbc-123.cloud.databricks.test/oidc/v1/token", endpoint)
}
