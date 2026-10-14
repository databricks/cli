package u2m

import (
	"errors"
	"fmt"
)

// ProfileChangedError is returned when a profile's cached token was created
// for different login settings than the ones the current command uses.
type ProfileChangedError struct {
	Profile string
}

func (e *ProfileChangedError) Error() string {
	return fmt.Sprintf("the cached login for profile %q does not match this command's settings. Run `databricks auth describe` with the same flags and environment variables to see where each setting comes from, or run `databricks auth login --profile %[1]s` if you changed the profile on purpose", e.Profile)
}

// ProfileChecksumMissingError is returned when a profile's cached token was
// created before the CLI stored login checksums.
type ProfileChecksumMissingError struct {
	Profile string
}

func (e *ProfileChecksumMissingError) Error() string {
	return fmt.Sprintf("cached credentials for profile %q were created by an older CLI version and cannot be checked against the profile. Run `databricks auth login --profile %[1]s` to sign in again", e.Profile)
}

// ErrMissingRefreshToken is returned when a token refresh is requested but the
// cached OAuth token does not include a refresh token.
var ErrMissingRefreshToken = errors.New("cached token has no refresh token")

// InvalidRefreshTokenError is returned from PersistentAuth's Token() and
// ForceRefreshToken() methods when a token refresh is attempted and the cached
// refresh token is invalid.
type InvalidRefreshTokenError struct {
	error
}
