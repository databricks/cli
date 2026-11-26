package skills

import (
	"fmt"
	"net/http"
)

// MaxFetchBodySize caps how much of a skills manifest or release lookup
// response is read.
const MaxFetchBodySize = 1 << 20

// ErrManifestTooLarge reports a manifest response larger than MaxFetchBodySize.
// The command layer classifies it like a manifest that does not parse.
var ErrManifestTooLarge = fmt.Errorf("skills manifest is larger than %d MiB", MaxFetchBodySize>>20)

// Stages of fetching skills content, recorded on FetchError so a failed
// release lookup can be told apart from a failed download.
const (
	// FetchStageRelease: resolving the latest skills release on api.github.com.
	FetchStageRelease = "release"
	// FetchStageManifest: downloading the skills manifest.
	FetchStageManifest = "manifest"
	// FetchStageFile: downloading one skill file.
	FetchStageFile = "file"
)

// FetchError reports that skills content could not be downloaded, or that a
// response arrived but could not be used (an unparsable manifest or release
// payload). It wraps the cause without changing its message; StatusCode is 0
// when no HTTP response arrived (DNS, proxy, TLS, timeout). A manifest that is
// not at the ref wraps clicompat.ErrNotFound, so clicompat.IsNotFoundError
// recognizes it like any other missing ref; check a 404 that way, not through
// StatusCode.
type FetchError struct {
	Stage      string
	StatusCode int
	// RateLimited is set when the server said the request exceeded a rate
	// limit, as opposed to an ordinary 403 from an access policy or proxy.
	RateLimited bool
	Err         error
}

func (e *FetchError) Error() string { return e.Err.Error() }
func (e *FetchError) Unwrap() error { return e.Err }

// NewHTTPFetchError builds the FetchError for a response with a non-OK status.
func NewHTTPFetchError(stage string, resp *http.Response, err error) *FetchError {
	return &FetchError{Stage: stage, StatusCode: resp.StatusCode, RateLimited: isRateLimited(resp), Err: err}
}

// isRateLimited reports whether a response is a rate-limit refusal. 429 always
// is; GitHub also answers 403 for exceeded limits, and then says so with
// x-ratelimit-remaining: 0 (primary limit) or retry-after (secondary limit).
// Any other 403 is an access failure.
// See https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api#exceeding-the-rate-limit.
func isRateLimited(resp *http.Response) bool {
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return true
	case http.StatusForbidden:
		return resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != ""
	default:
		return false
	}
}
