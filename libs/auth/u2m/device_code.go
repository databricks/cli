package u2m

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/httpclient"
	"golang.org/x/oauth2"
)

// confirmationCodeParam is the Databricks extension to the RFC 8628 token
// request. The user receives the confirmation code in the browser after
// approving the login, and the client submits it together with the device
// code instead of polling the token endpoint until the login is approved.
const confirmationCodeParam = "confirmation_code"

// deviceCodeGrantType is the grant type of the RFC 8628 token request.
const deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// ErrDeviceCodeNotSupported is returned when the workspace does not advertise
// a device authorization endpoint in its OIDC discovery document.
var ErrDeviceCodeNotSupported = errors.New("device code login is not enabled for this workspace")

// DeviceAuthorization holds what the user needs to approve a device code
// login from another device.
type DeviceAuthorization struct {
	// UserCode is the code the user enters at VerificationURI.
	UserCode string

	// VerificationURI is the page where the user approves the login.
	VerificationURI string
}

// maxConfirmationAttempts is how many confirmation codes the user can enter
// for a single device code before the login fails.
const maxConfirmationAttempts = 3

// ConfirmFunc shows the device authorization to the user and returns the
// confirmation code that the user received after approving the login. When
// the server rejects a confirmation code, it is called again with the
// server's error in lastErr so that the user can re-enter the code. lastErr
// is nil on the first call. attemptsLeft is the number of confirmation codes
// the user can still enter, including this one.
type ConfirmFunc func(ctx context.Context, da DeviceAuthorization, lastErr error, attemptsLeft int) (string, error)

// CheckDeviceCodeSupported returns ErrDeviceCodeNotSupported if the workspace
// does not advertise a device authorization endpoint in its OIDC discovery
// document. Callers use it to fail before prompting the user for anything.
func CheckDeviceCodeSupported(ctx context.Context, workspaceHost string) error {
	supplier := &BasicOAuthEndpointSupplier{Client: httpclient.NewApiClient(httpclient.ClientConfig{})}
	endpoints, err := supplier.GetWorkspaceOAuthEndpoints(ctx, workspaceHost)
	if err != nil {
		return fmt.Errorf("fetching OAuth endpoints: %w", err)
	}
	if endpoints.DeviceAuthorizationEndpoint == "" {
		return ErrDeviceCodeNotSupported
	}
	return nil
}

// DeviceCodeChallenge logs in using the Databricks variant of the RFC 8628
// device authorization grant. It requests a device authorization, calls
// confirm with the user code and verification URI, and exchanges the device
// code together with the returned confirmation code for a token. A rejected
// confirmation code leaves the device code valid, so the user can re-enter
// it up to maxConfirmationAttempts times. Only workspace hosts are supported.
// The caller is responsible for storing the returned token.
func (a *PersistentAuth) DeviceCodeChallenge(confirm ConfirmFunc) (*oauth2.Token, error) {
	if _, ok := a.oAuthArgument.(WorkspaceOAuthArgument); !ok {
		return nil, errors.New("device code login is only supported for workspace hosts")
	}
	cfg, err := a.oauth2Config()
	if err != nil {
		return nil, fmt.Errorf("fetching oauth config: %w", err)
	}
	if cfg.Endpoint.DeviceAuthURL == "" {
		return nil, ErrDeviceCodeNotSupported
	}

	ctx := a.setOAuthContext(a.ctx)
	da, err := cfg.DeviceAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("device authorization: %w", oauthError(err))
	}

	authorization := DeviceAuthorization{
		UserCode:        da.UserCode,
		VerificationURI: da.VerificationURI,
	}
	var lastErr error
	for attempt := range maxConfirmationAttempts {
		code, err := confirm(a.ctx, authorization, lastErr, maxConfirmationAttempts-attempt)
		if err != nil {
			return nil, err
		}

		t, err := a.exchangeDeviceCode(ctx, cfg, da.DeviceCode, normalizeCode(code))
		if err == nil {
			return t, nil
		}
		lastErr = oauthError(err)
		// The server answers a wrong confirmation code with HTTP 400 and
		// invalid_request. Any other error cannot be fixed by entering the
		// code again, including invalid_request with HTTP 429 when the server
		// rate limits the token requests for this device code.
		if status, errCode, _ := oauthErrorResponse(err); status != http.StatusBadRequest || errCode != "invalid_request" {
			return nil, fmt.Errorf("token exchange: %w", lastErr)
		}
	}
	return nil, fmt.Errorf("%w. Start login from scratch", lastErr)
}

// exchangeDeviceCode makes a single RFC 8628 device access token request.
// The server issues the token as soon as it receives a valid confirmation
// code, so unlike oauth2.Config.DeviceAccessToken this does not wait before
// the request or poll while the login is pending. It also does not enforce
// the expiry from the device authorization response, because the server
// extends the device code's lifetime when the user approves the login.
func (a *PersistentAuth) exchangeDeviceCode(ctx context.Context, cfg *oauth2.Config, deviceCode, confirmationCode string) (*oauth2.Token, error) {
	// No scope: the token gets the scopes of the device authorization
	// request, and the server ignores a scope sent here.
	form := url.Values{
		"grant_type":          {deviceCodeGrantType},
		"client_id":           {cfg.ClientID},
		"device_code":         {deviceCode},
		confirmationCodeParam: {confirmationCode},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		retrieveErr := &oauth2.RetrieveError{Response: resp, Body: body}
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			retrieveErr.ErrorCode = errResp.Error
			retrieveErr.ErrorDescription = errResp.ErrorDescription
		}
		return nil, retrieveErr
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return nil, errors.New("server response missing access_token")
	}
	t := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}
	if tokenResp.ExpiresIn > 0 {
		t.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	return t.WithExtra(raw), nil
}

// normalizeCode lets users enter a code the way it is displayed. Databricks
// codes consist of 8 uppercase alphanumeric characters and are compared
// case-sensitively, but are often shown as two groups of four separated by a
// dash (e.g. FTFA-QQKY).
func normalizeCode(code string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(strings.ToUpper(strings.TrimSpace(code)))
}

// oauthError extracts the RFC 6749 §5.2 error response from err so that users
// see the server's explanation instead of a raw HTTP response body.
func oauthError(err error) error {
	_, code, description := oauthErrorResponse(err)
	if code == "" {
		return err
	}
	return fmt.Errorf("%s (error code: %s)", description, code)
}

// oauthErrorResponse returns the HTTP status code, and the error code and
// description of the RFC 6749 §5.2 error response carried by err. The status
// is 0 if err does not carry an HTTP response, and the code is empty if err
// does not carry an error response.
func oauthErrorResponse(err error) (status int, code, description string) {
	// The default RoundTripper of httpclient.ApiClient returns an error for
	// non-2xx responses, so the oauth2 library never parses the body itself.
	if httpErr, ok := errors.AsType[*httpclient.HttpError](err); ok {
		var resp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal([]byte(httpErr.Message), &resp) != nil {
			return httpErr.StatusCode, "", ""
		}
		return httpErr.StatusCode, resp.Error, resp.ErrorDescription
	}
	if retrieveErr, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		return retrieveErr.Response.StatusCode, retrieveErr.ErrorCode, retrieveErr.ErrorDescription
	}
	return 0, "", ""
}
