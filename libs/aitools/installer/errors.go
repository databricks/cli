package installer

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrNothingInstalled reports that update or uninstall found no install state
// in the scope. The command layer matches it with errors.Is to tell "nothing to
// act on" apart from a failure.
var ErrNothingInstalled = errors.New("no skills or plugins installed")

// ErrUnknownAgent reports a plugin record in the install state for an agent
// this CLI does not know (e.g. one written by a newer CLI). Update and
// uninstall leave such a record alone and report it as a failed plugin.
var ErrUnknownAgent = errors.New("unknown agent")

// unknownAgentInStateError returns ErrUnknownAgent for the agent name recorded
// in the state.
func unknownAgentInStateError(name string) error {
	return fmt.Errorf("%w %q in state", ErrUnknownAgent, name)
}

// ErrLegacyInstall reports skills from a previous install that predates state
// tracking: update and uninstall need the install state, so they ask the user
// to run install first.
var ErrLegacyInstall = errors.New("found skills from a previous install without state tracking")

// SkillError reports that a skill named via --skills could not be resolved from
// the manifest or, for uninstall, is not installed. Reason drives telemetry
// categorization (the command layer maps it with errors.AsType); Detail is the
// human-readable remainder of the message. Building the message from fields
// keeps a classification tag out of the user-facing string, so the error is
// stated exactly once.
type SkillError struct {
	Skill  string
	Reason string
	Detail string
}

// Reasons a --skills entry can fail to resolve.
const (
	// ReasonSkillNotFound: the named skill is absent from the resolved manifest
	// or, for uninstall, from the install state.
	ReasonSkillNotFound = "skill-not-found"
	// ReasonVersionIncompatible: the skill requires a newer CLI than the one running.
	ReasonVersionIncompatible = "version-incompatible"
	// ReasonExperimentalSkill: the named skill is experimental and --experimental was not set.
	ReasonExperimentalSkill = "experimental-skill"
)

func (e *SkillError) Error() string {
	// Detail is the human-readable remainder; fall back to Reason when it is empty
	// so the message stays self-describing (mirrors BlockedError.Error()).
	if e.Detail != "" {
		return fmt.Sprintf("skill %q %s", e.Skill, e.Detail)
	}
	return fmt.Sprintf("skill %q %s", e.Skill, e.Reason)
}

// maxFetchBodySize caps how much of a skills manifest or release lookup
// response is read.
const maxFetchBodySize = 1 << 20

// ErrManifestTooLarge reports a manifest response larger than maxFetchBodySize.
// The command layer classifies it like a manifest that does not parse.
var ErrManifestTooLarge = fmt.Errorf("skills manifest is larger than %d MiB", maxFetchBodySize>>20)

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

// newHTTPFetchError builds the FetchError for a response with a non-OK status.
func newHTTPFetchError(stage string, resp *http.Response, err error) *FetchError {
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

// StateError reports that the install state file exists but its content is
// invalid. Path is the state file that failed to parse.
// A file that cannot be read is not a StateError: that is a filesystem problem
// (usually permissions), and moving the file aside would not fix it.
type StateError struct {
	Path string
	Err  error
}

func (e *StateError) Error() string { return e.Err.Error() }
func (e *StateError) Unwrap() error { return e.Err }
