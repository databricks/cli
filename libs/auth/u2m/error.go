package u2m

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go/httpclient"
	"golang.org/x/oauth2"
)

// ErrMissingRefreshToken is returned when a token refresh is requested but the
// cached OAuth token does not include a refresh token.
var ErrMissingRefreshToken = errors.New("cached token has no refresh token")

// InvalidRefreshTokenError is returned from PersistentAuth's Token() and
// ForceRefreshToken() methods when a token refresh is attempted and the cached
// refresh token is invalid.
type InvalidRefreshTokenError struct {
	error
}

// oauthError extracts the RFC 6749 §5.2 error response from err so that users
// see the server's explanation instead of a raw HTTP response body.
func oauthError(err error) error {
	_, code, description := oauthErrorResponse(err)
	if code == "" {
		return err
	}
	return &oauthResponseError{code: code, description: description, err: err}
}

// oauthResponseError shows an RFC 6749 §5.2 error response and keeps the
// error that carried it, so that callers can still inspect the HTTP response.
type oauthResponseError struct {
	code        string
	description string
	err         error
}

func (e *oauthResponseError) Error() string {
	return fmt.Sprintf("%s (error code: %s)", e.description, e.code)
}

func (e *oauthResponseError) Unwrap() error {
	return e.err
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
