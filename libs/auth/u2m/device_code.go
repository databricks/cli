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

	"golang.org/x/oauth2"
)

// confirmationCodeParam is the Databricks extension to the RFC 8628 token
// request. The user receives the confirmation code in the browser after
// approving the login, and the client submits it together with the device
// code instead of polling the token endpoint until the login is approved.
const confirmationCodeParam = "confirmation_code"

// deviceCodeGrantType is the grant type of the RFC 8628 token request.
const deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// invalidDeviceCodeDescription is the error_description the server sends with
// HTTP 400 and invalid_request for a device code it does not know, e.g. one
// it discarded after the login expired.
const invalidDeviceCodeDescription = "Invalid device code"

// ErrDeviceCodeNotSupported is returned when the workspace does not advertise
// a device authorization endpoint in its OIDC discovery document.
var ErrDeviceCodeNotSupported = errors.New("device code login is not enabled for this workspace")

// ErrInvalidConfirmationCode matches the token exchange error for a
// confirmation code the server rejects. The device code stays valid, so the
// exchange can be retried with the right code.
var ErrInvalidConfirmationCode = errors.New("invalid confirmation code")

// ErrInvalidDeviceCode matches the token exchange error for a device code the
// server does not know, e.g. one it discarded after the login expired. The
// login has to be started again.
var ErrInvalidDeviceCode = errors.New("invalid device code")

// DeviceAuthorization holds what the user needs to approve a device code
// login from another device, and the device code to exchange afterwards.
type DeviceAuthorization struct {
	// UserCode is the code the user enters at VerificationURI.
	UserCode string

	// VerificationURI is the page where the user approves the login.
	VerificationURI string

	// DeviceCode identifies the login in the token exchange.
	DeviceCode string
}

// deviceCodeConfig returns the OAuth config for device code login, or an
// error if the host does not support it.
func (a *PersistentAuth) deviceCodeConfig(ctx context.Context) (*oauth2.Config, error) {
	if _, ok := a.oAuthArgument.(WorkspaceOAuthArgument); !ok {
		return nil, errors.New("device code login is only supported for workspace hosts")
	}
	// Resource indicators are only sent on the browser authorization request,
	// so the token would not be scoped to them.
	if len(a.resources) > 0 {
		return nil, errors.New("device code login does not support resource indicators")
	}
	cfg, err := a.oauth2Config(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching oauth config: %w", err)
	}
	if cfg.Endpoint.DeviceAuthURL == "" {
		return nil, ErrDeviceCodeNotSupported
	}
	return cfg, nil
}

// DeviceCodeLogin makes the requests of a login using the Databricks variant
// of the RFC 8628 device authorization grant. Get one with
// PersistentAuth.NewDeviceCodeLogin.
type DeviceCodeLogin struct {
	auth *PersistentAuth
	cfg  *oauth2.Config
}

// NewDeviceCodeLogin fetches the OAuth endpoints for a device code login, or
// returns an error if the host does not support it. Only workspace hosts are
// supported.
func (a *PersistentAuth) NewDeviceCodeLogin(ctx context.Context) (*DeviceCodeLogin, error) {
	cfg, err := a.deviceCodeConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &DeviceCodeLogin{auth: a, cfg: cfg}, nil
}

// RequestDeviceAuthorization starts the login. The caller shows the user code
// and the verification URI to the user and later passes the device code to
// RequestDeviceAccessToken.
func (l *DeviceCodeLogin) RequestDeviceAuthorization(ctx context.Context) (*DeviceAuthorization, error) {
	da, err := l.auth.requestDeviceAuthorization(ctx, l.cfg)
	if err != nil {
		return nil, fmt.Errorf("device authorization: %w", oauthError(err))
	}
	return &DeviceAuthorization{
		UserCode:        da.UserCode,
		VerificationURI: da.VerificationURI,
		DeviceCode:      da.DeviceCode,
	}, nil
}

// RequestDeviceAccessToken completes the login with a single token request.
// The confirmation code is accepted the way the browser displays it. An error
// matches ErrInvalidConfirmationCode or ErrInvalidDeviceCode when the server
// rejects one of the codes. A rejected confirmation code leaves the device
// code valid, so the exchange can be retried with the right code. The caller
// is responsible for storing the returned token.
func (l *DeviceCodeLogin) RequestDeviceAccessToken(ctx context.Context, deviceCode, confirmationCode string) (*oauth2.Token, error) {
	t, err := l.auth.exchangeDeviceCode(ctx, l.cfg, deviceCode, NormalizeCode(confirmationCode))
	if err == nil {
		return t, nil
	}
	// The server answers a wrong confirmation code with HTTP 400 and
	// invalid_request. Any other error cannot be fixed by entering the
	// code again, including invalid_request with HTTP 429 when the server
	// rate limits the token requests for this device code. An unknown
	// device code also gets HTTP 400 and invalid_request, and only the
	// description tells it apart. If the description changes, the error is
	// reported as a wrong confirmation code.
	status, errCode, description := oauthErrorResponse(err)
	err = oauthError(err)
	switch {
	case status != http.StatusBadRequest || errCode != "invalid_request":
		return nil, err
	case description == invalidDeviceCodeDescription:
		return nil, &codeRejectedError{err: err, sentinel: ErrInvalidDeviceCode}
	default:
		return nil, &codeRejectedError{err: err, sentinel: ErrInvalidConfirmationCode}
	}
}

// codeRejectedError shows the server's error for a rejected code and matches
// sentinel with errors.Is.
type codeRejectedError struct {
	err      error
	sentinel error
}

func (e *codeRejectedError) Error() string {
	return e.err.Error()
}

func (e *codeRejectedError) Unwrap() []error {
	return []error{e.err, e.sentinel}
}

// requestDeviceAuthorization makes the RFC 8628 device authorization request.
// Unlike oauth2.Config.DeviceAuth, it sends the request with ctx, so that the
// login timeout and cancellation stop a stalled request.
func (a *PersistentAuth) requestDeviceAuthorization(ctx context.Context, cfg *oauth2.Config) (*oauth2.DeviceAuthResponse, error) {
	body, err := a.postForm(ctx, cfg.Endpoint.DeviceAuthURL, url.Values{
		"client_id": {cfg.ClientID},
		"scope":     {strings.Join(cfg.Scopes, " ")},
	})
	if err != nil {
		return nil, err
	}
	var da oauth2.DeviceAuthResponse
	if err := json.Unmarshal(body, &da); err != nil {
		return nil, fmt.Errorf("parsing device authorization response: %w", err)
	}
	// The user needs user_code and verification_uri to approve the login, and
	// the token request needs device_code.
	switch {
	case da.DeviceCode == "":
		return nil, errors.New("server response missing device_code")
	case da.UserCode == "":
		return nil, errors.New("server response missing user_code")
	case da.VerificationURI == "":
		return nil, errors.New("server response missing verification_uri")
	}
	return &da, nil
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
	body, err := a.postForm(ctx, cfg.Endpoint.TokenURL, url.Values{
		"grant_type":          {deviceCodeGrantType},
		"client_id":           {cfg.ClientID},
		"device_code":         {deviceCode},
		confirmationCodeParam: {confirmationCode},
	})
	if err != nil {
		return nil, err
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

// postForm sends a form-encoded OAuth request and returns the body of a
// successful response. An error response is returned as an
// *oauth2.RetrieveError with the RFC 6749 §5.2 error fields.
func (a *PersistentAuth) postForm(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
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
	return body, nil
}

// NormalizeCode lets users enter a code the way it is displayed. Databricks
// codes consist of 8 uppercase alphanumeric characters and are compared
// case-sensitively, but are often shown as two groups of four separated by a
// dash (e.g. FTFA-QQKY).
func NormalizeCode(code string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(strings.ToUpper(strings.TrimSpace(code)))
}
