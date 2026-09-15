package server_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/server"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCredentials struct{ authenticate func(*http.Request) error }

func (c testCredentials) Name() string { return "test" }

func (c testCredentials) Configure(context.Context, *config.Config) (credentials.CredentialsProvider, error) {
	return credentials.CredentialsProviderFn(c.authenticate), nil
}

func TestWorkspaceToken(t *testing.T) {
	for _, header := range []string{"Bearer token-one", "Basic private-credential", "Bearer ", ""} {
		t.Run(header, func(t *testing.T) {
			c, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host: "https://workspace.test", HostMetadataResolver: noHostMetadata, Token: "stale-config-token",
				Credentials: testCredentials{authenticate: func(r *http.Request) error {
					r.Header.Set("Authorization", header)
					return nil
				}},
			})
			require.NoError(t, err)
			token, err := server.WorkspaceToken(c)(t.Context())
			if header == "Bearer token-one" {
				require.NoError(t, err)
				assert.Equal(t, "token-one", token)
				return
			}
			require.ErrorContains(t, err, "do not provide a bearer token")
			assert.NotContains(t, err.Error(), "private-credential")
			assert.Empty(t, token)
		})
	}
}

func TestWorkspaceTokenRefreshAndFailure(t *testing.T) {
	value := "first"
	var authErr error
	c, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host: "https://workspace.test", HostMetadataResolver: noHostMetadata,
		Credentials: testCredentials{authenticate: func(r *http.Request) error {
			r.Header.Set("Authorization", "Bearer "+value)
			return authErr
		}},
	})
	require.NoError(t, err)
	token := server.WorkspaceToken(c)
	first, err := token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "first", first)
	value = "second"
	second, err := token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "second", second)
	authErr = errors.New("credentials unavailable")
	_, err = token(t.Context())
	require.ErrorIs(t, err, authErr)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFuseUserID(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body, want string
	}{
		{name: "available", status: http.StatusOK, body: `{"id":"12345"}`, want: "12345"},
		{name: "forbidden", status: http.StatusForbidden, body: `{"message":"forbidden"}`},
		{name: "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c, err := databricks.NewWorkspaceClient(&databricks.Config{
					Host: "https://workspace.test", HostMetadataResolver: noHostMetadata, Token: "test-token",
					HTTPTransport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						assert.Equal(t, "/api/2.0/preview/scim/v2/Me", r.URL.Path)
						if tc.status == 0 {
							<-r.Context().Done()
							return nil, r.Context().Err()
						}
						return &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
					}),
				})
				require.NoError(t, err)
				start := time.Now()
				assert.Equal(t, tc.want, server.FuseUserID(t.Context(), c))
				assert.LessOrEqual(t, time.Since(start), 2*time.Second)
			})
		})
	}
}

func noHostMetadata(context.Context, string) (*config.HostMetadata, error) { return nil, nil }
