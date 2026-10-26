package testserver

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
)

const (
	deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

	// Device code login is approved on another device, which acceptance tests
	// cannot drive, so every device authorization is treated as approved with
	// a fixed confirmation code that tests can type into the CLI.
	fakeDeviceUserCode         = "WDJBMJHT"
	fakeDeviceConfirmationCode = "FTFAQQKY"

	// Like the real server, only allowlisted OAuth clients can use device code
	// login. The fake allows only the CLI's default client.
	fakeDeviceClientID = "databricks-cli"
)

// FakeOidc holds OAuth state for acceptance tests.
type FakeOidc struct {
	url string

	mu sync.Mutex

	// deviceCodes maps device codes that have not been exchanged yet to
	// whether offline_access was requested for them.
	deviceCodes    map[string]bool
	lastDeviceCode int
}

func (s *FakeOidc) OidcEndpoints() Response {
	return Response{
		Body: map[string]string{
			"authorization_endpoint":        s.url + "/oidc/v1/authorize",
			"token_endpoint":                s.url + "/oidc/v1/token",
			"device_authorization_endpoint": s.url + "/oidc/v1/device/auth",
		},
	}
}

// OidcDeviceAuth implements the RFC 8628 device authorization endpoint.
func (s *FakeOidc) OidcDeviceAuth(req Request) Response {
	form, err := url.ParseQuery(string(req.Body))
	if err != nil || form.Get("client_id") == "" {
		return oauthErrorResponse("invalid_request", "Bad Request")
	}
	if form.Get("client_id") != fakeDeviceClientID {
		return oauthErrorResponse("unauthorized_client", "Client is not allowed to use device code login")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deviceCodes == nil {
		s.deviceCodes = map[string]bool{}
	}
	s.lastDeviceCode++
	deviceCode := "device-code-" + strconv.Itoa(s.lastDeviceCode)
	s.deviceCodes[deviceCode] = slices.Contains(strings.Fields(form.Get("scope")), "offline_access")

	return Response{
		Body: map[string]any{
			"device_code":      deviceCode,
			"user_code":        fakeDeviceUserCode,
			"verification_uri": s.url + "/device/verification",
			"expires_in":       600,
			"interval":         0,
		},
	}
}

// deviceCodeToken exchanges a device code and confirmation code for a token.
// Like the real server, a wrong confirmation code leaves the device code
// valid, and a successful exchange consumes it.
func (s *FakeOidc) deviceCodeToken(form url.Values) Response {
	s.mu.Lock()
	defer s.mu.Unlock()
	deviceCode := form.Get("device_code")
	offlineAccess, ok := s.deviceCodes[deviceCode]
	if !ok {
		return oauthErrorResponse("invalid_request", "Invalid device code")
	}
	if form.Get("confirmation_code") != fakeDeviceConfirmationCode {
		return oauthErrorResponse("invalid_request", "Invalid confirmation code")
	}
	delete(s.deviceCodes, deviceCode)

	body := map[string]any{
		"access_token": "oauth-token",
		"expires_in":   3600,
		"scope":        "all-apis",
		"token_type":   "Bearer",
	}
	if offlineAccess {
		body["refresh_token"] = "oauth-refresh-token"
	}
	return Response{Body: body}
}

func oauthErrorResponse(code, description string) Response {
	return Response{
		StatusCode: http.StatusBadRequest,
		Body: map[string]string{
			"error":             code,
			"error_description": description,
		},
	}
}

func (s *FakeOidc) OidcAuthorize(req Request) Response {
	redirectURI, err := url.Parse(req.URL.Query().Get("redirect_uri"))
	if err != nil {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       err.Error(),
		}
	}

	// Compile query parameters for the redirect URL.
	q := make(url.Values)

	// Include an opaque authorization code that will be used to exchange for a token.
	q.Set("code", "oauth-code")

	// Include the state parameter from the original request.
	q.Set("state", req.URL.Query().Get("state"))

	// Update the redirect URI with the new query parameters.
	redirectURI.RawQuery = q.Encode()

	return Response{
		StatusCode: http.StatusMovedPermanently,
		Headers: map[string][]string{
			"Location": {redirectURI.String()},
		},
	}
}

func (s *FakeOidc) OidcToken(req Request) Response {
	form, err := url.ParseQuery(string(req.Body))
	if err == nil && form.Get("grant_type") == deviceCodeGrantType {
		return s.deviceCodeToken(form)
	}
	return Response{
		Body: map[string]string{
			"access_token": "oauth-token",
			"expires_in":   "3600",
			"scope":        "all-apis",
			"token_type":   "Bearer",
		},
	}
}
