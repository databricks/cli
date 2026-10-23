package u2m

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDeviceCode       = "device-code-123"
	testUserCode         = "WDJBMJHT"
	testConfirmationCode = "FTFAQQKY"
)

// deviceCodeEndpointSupplier serves workspace OIDC endpoints from a test
// server, optionally advertising the device authorization endpoint.
type deviceCodeEndpointSupplier struct {
	MockOAuthEndpointSupplier
	host                string
	deviceAuthAdvertise bool
}

func (s deviceCodeEndpointSupplier) GetWorkspaceOAuthEndpoints(_ context.Context, _ string) (*OAuthAuthorizationServer, error) {
	endpoints := &OAuthAuthorizationServer{
		AuthorizationEndpoint: s.host + "/oidc/v1/authorize",
		TokenEndpoint:         s.host + "/oidc/v1/token",
	}
	if s.deviceAuthAdvertise {
		endpoints.DeviceAuthorizationEndpoint = s.host + "/oidc/v1/device/auth"
	}
	return endpoints, nil
}

type fakeDeviceServer struct {
	deviceAuthStatus int
	deviceAuthBody   string
	tokenStatus      int
	tokenBody        string
	expiresIn        int

	tokenRequests atomic.Int32
}

func (f *fakeDeviceServer) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oidc/v1/device/auth", func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "databricks-cli", r.PostForm.Get("client_id"))
		assert.Equal(t, "offline_access all-apis", r.PostForm.Get("scope"))
		w.Header().Set("Content-Type", "application/json")
		if f.deviceAuthStatus != 0 {
			w.WriteHeader(f.deviceAuthStatus)
			_, _ = w.Write([]byte(f.deviceAuthBody))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code":      testDeviceCode,
			"user_code":        testUserCode,
			"verification_uri": "https://workspace.test/device/verification",
			"expires_in":       f.expiresIn,
			"interval":         0,
		})
	})
	mux.HandleFunc("POST /oidc/v1/token", func(w http.ResponseWriter, r *http.Request) {
		f.tokenRequests.Add(1)
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "urn:ietf:params:oauth:grant-type:device_code", r.PostForm.Get("grant_type"))
		assert.Equal(t, "databricks-cli", r.PostForm.Get("client_id"))
		assert.Equal(t, testDeviceCode, r.PostForm.Get("device_code"))
		w.Header().Set("Content-Type", "application/json")
		if f.tokenStatus != 0 {
			w.WriteHeader(f.tokenStatus)
			_, _ = w.Write([]byte(f.tokenBody))
			return
		}
		if r.PostForm.Get("confirmation_code") != testConfirmationCode {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "invalid_request", "error_description": "Invalid confirmation code"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	})
	return mux
}

func newDeviceCodeAuth(t *testing.T, f *fakeDeviceServer, advertise bool, opts ...PersistentAuthOption) *PersistentAuth {
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	arg, err := NewBasicWorkspaceOAuthArgument("https://workspace.test")
	require.NoError(t, err)
	p, err := NewPersistentAuth(t.Context(), append([]PersistentAuthOption{
		WithOAuthArgument(arg),
		WithOAuthEndpointSupplier(deviceCodeEndpointSupplier{host: srv.URL, deviceAuthAdvertise: advertise}),
	}, opts...)...)
	require.NoError(t, err)
	t.Cleanup(func() { p.Close() })
	return p
}

func TestDeviceCodeChallenge(t *testing.T) {
	f := &fakeDeviceServer{expiresIn: 600}
	p := newDeviceCodeAuth(t, f, true)

	var got DeviceAuthorization
	tok, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, da DeviceAuthorization, _ error, _ int) (string, error) {
		got = da
		return " ftfa-qqky\t", nil
	})
	require.NoError(t, err)

	assert.Equal(t, DeviceAuthorization{
		UserCode:        testUserCode,
		VerificationURI: "https://workspace.test/device/verification",
	}, got)
	assert.Equal(t, "access-token", tok.AccessToken)
	assert.Equal(t, "refresh-token", tok.RefreshToken)
	assert.WithinDuration(t, time.Now().Add(time.Hour), tok.Expiry, time.Minute)
	assert.Equal(t, int32(1), f.tokenRequests.Load())
}

func TestDeviceCodeChallenge_ConfirmationAfterInitialExpiry(t *testing.T) {
	// The server extends the device code's lifetime once the user approves
	// the login, so a confirmation code entered after the initial expires_in
	// must still be exchanged.
	f := &fakeDeviceServer{expiresIn: 1}
	p := newDeviceCodeAuth(t, f, true)

	tok, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, _ int) (string, error) {
		time.Sleep(1100 * time.Millisecond)
		return testConfirmationCode, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "access-token", tok.AccessToken)
}

func TestDeviceCodeChallenge_Errors(t *testing.T) {
	tests := []struct {
		name    string
		server  *fakeDeviceServer
		wantErr string
	}{
		{
			name: "device authorization rejected",
			server: &fakeDeviceServer{
				deviceAuthStatus: http.StatusBadRequest,
				deviceAuthBody:   `{"error": "unauthorized_client", "error_description": "Client is not allowed to use device code"}`,
			},
			wantErr: "device authorization: Client is not allowed to use device code (error code: unauthorized_client)",
		},
		{
			name: "token request rejected",
			server: &fakeDeviceServer{
				tokenStatus: http.StatusBadRequest,
				tokenBody:   `{"error": "invalid_client", "error_description": "Client is not allowed to use device code"}`,
			},
			wantErr: "token exchange: Client is not allowed to use device code (error code: invalid_client)",
		},
		{
			name: "token request rate limited",
			server: &fakeDeviceServer{
				tokenStatus: http.StatusTooManyRequests,
				tokenBody:   `{"error": "invalid_request", "error_description": "Device grant rate limit exceeded"}`,
			},
			wantErr: "token exchange: Device grant rate limit exceeded (error code: invalid_request)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.server.expiresIn = 600
			p := newDeviceCodeAuth(t, tc.server, true)
			_, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, _ int) (string, error) {
				return testConfirmationCode, nil
			})
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestDeviceCodeChallenge_RetriesWrongConfirmationCode(t *testing.T) {
	f := &fakeDeviceServer{expiresIn: 600}
	p := newDeviceCodeAuth(t, f, true)

	var lastErrs []string
	tok, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, lastErr error, _ int) (string, error) {
		if lastErr == nil {
			lastErrs = append(lastErrs, "")
			return "AAAAAAAA", nil
		}
		lastErrs = append(lastErrs, lastErr.Error())
		return testConfirmationCode, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "access-token", tok.AccessToken)
	assert.Equal(t, []string{"", "Invalid confirmation code (error code: invalid_request)"}, lastErrs)
	assert.Equal(t, int32(2), f.tokenRequests.Load())
}

func TestDeviceCodeChallenge_TooManyWrongConfirmationCodes(t *testing.T) {
	f := &fakeDeviceServer{expiresIn: 600}
	p := newDeviceCodeAuth(t, f, true)

	var attemptsLeft []int
	_, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, left int) (string, error) {
		attemptsLeft = append(attemptsLeft, left)
		return "AAAAAAAA", nil
	})
	assert.EqualError(t, err, "Invalid confirmation code (error code: invalid_request). Start login from scratch")
	assert.Equal(t, []int{3, 2, 1}, attemptsLeft)
	assert.Equal(t, int32(maxConfirmationAttempts), f.tokenRequests.Load())
}

func TestDeviceCodeChallenge_ConfirmErrorOnRetry(t *testing.T) {
	f := &fakeDeviceServer{expiresIn: 600}
	p := newDeviceCodeAuth(t, f, true)
	confirmErr := errors.New("no input")

	_, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, lastErr error, _ int) (string, error) {
		if lastErr != nil {
			return "", confirmErr
		}
		return "AAAAAAAA", nil
	})
	assert.ErrorIs(t, err, confirmErr)
	assert.Equal(t, int32(1), f.tokenRequests.Load())
}

func TestDeviceCodeChallenge_DoesNotPoll(t *testing.T) {
	// A plain HTTP client hands the error response to the CLI as is, rather
	// than turning it into an HttpError. A pending login must fail right away
	// instead of being polled until it is approved.
	f := &fakeDeviceServer{
		expiresIn:   600,
		tokenStatus: http.StatusBadRequest,
		tokenBody:   `{"error": "authorization_pending", "error_description": "Login is not approved yet"}`,
	}
	p := newDeviceCodeAuth(t, f, true, WithHttpClient(&http.Client{}))

	_, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, _ int) (string, error) {
		return testConfirmationCode, nil
	})
	assert.EqualError(t, err, "token exchange: Login is not approved yet (error code: authorization_pending)")
	assert.Equal(t, int32(1), f.tokenRequests.Load())
}

func TestDeviceCodeChallenge_MissingAccessToken(t *testing.T) {
	f := &fakeDeviceServer{
		expiresIn:   600,
		tokenStatus: http.StatusOK,
		tokenBody:   `{"token_type": "Bearer"}`,
	}
	p := newDeviceCodeAuth(t, f, true)

	_, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, _ int) (string, error) {
		return testConfirmationCode, nil
	})
	assert.EqualError(t, err, "token exchange: server response missing access_token")
}

func TestDeviceCodeChallenge_ConfirmError(t *testing.T) {
	f := &fakeDeviceServer{expiresIn: 600}
	p := newDeviceCodeAuth(t, f, true)
	confirmErr := errors.New("no input")

	_, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, _ int) (string, error) {
		return "", confirmErr
	})
	assert.ErrorIs(t, err, confirmErr)
	assert.Equal(t, int32(0), f.tokenRequests.Load())
}

func TestDeviceCodeChallenge_NotAdvertised(t *testing.T) {
	p := newDeviceCodeAuth(t, &fakeDeviceServer{}, false)

	_, err := p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, _ int) (string, error) {
		t.Fatal("confirm must not be called")
		return "", nil
	})
	assert.ErrorIs(t, err, ErrDeviceCodeNotSupported)
}

func TestDeviceCodeChallenge_AccountHost(t *testing.T) {
	arg, err := NewBasicAccountOAuthArgument("https://accounts.cloud.databricks.test", "xyz")
	require.NoError(t, err)
	p, err := NewPersistentAuth(t.Context(),
		WithOAuthArgument(arg),
		WithOAuthEndpointSupplier(MockOAuthEndpointSupplier{}),
	)
	require.NoError(t, err)
	defer p.Close()

	_, err = p.DeviceCodeChallenge(t.Context(), func(_ context.Context, _ DeviceAuthorization, _ error, _ int) (string, error) {
		t.Fatal("confirm must not be called")
		return "", nil
	})
	assert.EqualError(t, err, "device code login is only supported for workspace hosts")
}

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"FTFAQQKY", "FTFAQQKY"},
		{"FTFA-QQKY", "FTFAQQKY"},
		{"ftfa-qqky", "FTFAQQKY"},
		{"  FTFA QQKY \n", "FTFAQQKY"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, normalizeCode(tc.in), tc.in)
	}
}

func TestCheckDeviceCodeSupported(t *testing.T) {
	tests := []struct {
		name      string
		advertise bool
		wantErr   error
	}{
		{name: "advertised", advertise: true},
		{name: "not advertised", wantErr: ErrDeviceCodeNotSupported},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newDeviceCodeAuth(t, &fakeDeviceServer{}, tc.advertise)
			assert.ErrorIs(t, p.CheckDeviceCodeSupported(), tc.wantErr)
		})
	}
}
