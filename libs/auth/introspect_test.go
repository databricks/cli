package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntrospectToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"principal_context": {
				"authentication_scope": {
					"account_id": "a1b1c234-5678-90ab-cdef-1234567890ab",
					"workspace_id": 2548836972759138
				}
			}
		}`))
	}))
	defer server.Close()

	result, err := IntrospectToken(t.Context(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "a1b1c234-5678-90ab-cdef-1234567890ab", result.AccountID)
	assert.Equal(t, "2548836972759138", result.WorkspaceID)
}

func TestIntrospectToken_ZeroWorkspaceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"principal_context": {
				"authentication_scope": {
					"account_id": "abc-123",
					"workspace_id": 0
				}
			}
		}`))
	}))
	defer server.Close()

	result, err := IntrospectToken(t.Context(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", result.AccountID)
	assert.Empty(t, result.WorkspaceID)
}

func TestIntrospectToken_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := IntrospectToken(t.Context(), server.URL, "test-token")
	assert.ErrorContains(t, err, "status 403")
}

func TestIntrospectToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := IntrospectToken(t.Context(), server.URL, "test-token")
	assert.ErrorContains(t, err, "status 500")
}

func TestIntrospectToken_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	_, err := IntrospectToken(t.Context(), server.URL, "test-token")
	assert.ErrorContains(t, err, "decoding introspection response")
}

func TestIntrospectToken_VerifyAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-secret-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"principal_context":{"authentication_scope":{"account_id":"x","workspace_id":1}}}`))
	}))
	defer server.Close()

	_, err := IntrospectToken(t.Context(), server.URL, "my-secret-token")
	require.NoError(t, err)
}

func TestIntrospectToken_VerifyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/tokens/introspect", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"principal_context":{"authentication_scope":{"account_id":"x","workspace_id":1}}}`))
	}))
	defer server.Close()

	_, err := IntrospectToken(t.Context(), server.URL, "token")
	require.NoError(t, err)
}
