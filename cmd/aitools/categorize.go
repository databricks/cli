package aitools

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"os"
	"syscall"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// categorizedError tags an error with a telemetry category at the site that
// knows what failed (flag validation, agent lookup, a prompt). The tag wins
// over the standard library types. Error() and Unwrap() pass through, so the
// user-visible message is unchanged.
type categorizedError struct {
	err      error
	category protos.AitoolsErrorCategory
}

func (e *categorizedError) Error() string { return e.err.Error() }
func (e *categorizedError) Unwrap() error { return e.err }

// withCategory tags a non-nil err with category.
func withCategory(err error, category protos.AitoolsErrorCategory) error {
	return &categorizedError{err: err, category: category}
}

// promptError tags a non-nil prompt failure. A user abort (Ctrl-C) keeps its own
// classification so the funnel reports it as CANCELLED, not as a failure.
// Any other error of a prompt step is PROMPT_FAILED, even one with an OS
// error type: the prompt library wraps a failure to open the terminal as an
// *fs.PathError, and FILESYSTEM would point at the skill files instead.
func promptError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return err
	}
	return withCategory(err, protos.AitoolsErrorCategoryPromptFailed)
}

// classifyInstallError maps an aitools error to its telemetry category without
// reading any error text. Typed installer errors win, then categories tagged at
// the failure site, then standard library error types.
func classifyInstallError(err error) protos.AitoolsErrorCategory {
	if err == nil {
		return protos.AitoolsErrorCategoryUnspecified
	}

	if blocked, ok := errors.AsType[*installer.BlockedError](err); ok {
		return blockedErrorCategory(blocked)
	}
	if skill, ok := errors.AsType[*installer.SkillError](err); ok {
		return skillErrorCategory(skill)
	}
	// FetchError is checked before the standard library types: a response
	// that arrived carries a status the net error types know nothing about.
	if fetch, ok := errors.AsType[*installer.FetchError](err); ok {
		return fetchErrorCategory(fetch)
	}
	if _, ok := errors.AsType[*installer.StateError](err); ok {
		return protos.AitoolsErrorCategoryStateInvalid
	}
	if tagged, ok := errors.AsType[*categorizedError](err); ok {
		return tagged.category
	}
	if errors.Is(err, installer.ErrUnknownAgent) {
		return protos.AitoolsErrorCategoryUnknownAgent
	}
	// Skills on disk without the install state that update and uninstall need.
	if errors.Is(err, installer.ErrLegacyInstall) {
		return protos.AitoolsErrorCategoryStateInvalid
	}
	return stdlibErrorCategory(err)
}

// stdlibErrorCategory classifies errors by their standard library type, which
// the installer wraps with %w: filesystem errors from writing skills or the
// install state, and network and timeout errors, including those of a
// FetchError that got no response. JSON decode errors stay UNCATEGORIZED: the
// manifest, the release payload and the install state all decode JSON, so only
// typed installer errors can tell MANIFEST_INVALID from STATE_INVALID.
func stdlibErrorCategory(err error) protos.AitoolsErrorCategory {
	switch {
	case errors.Is(err, huh.ErrUserAborted):
		return protos.AitoolsErrorCategoryUserAborted
	case errors.Is(err, context.DeadlineExceeded):
		return protos.AitoolsErrorCategoryTimeout
	case errors.Is(err, context.Canceled):
		return protos.AitoolsErrorCategoryCommandInterrupted
	}
	// I/O deadlines (os.ErrDeadlineExceeded) are net.Errors whose Timeout()
	// is true. syscall.Errno also implements net.Error, so a bare OS error code
	// (from a file operation) is not evidence of a network failure.
	if netErr, ok := errors.AsType[net.Error](err); ok && !isErrno(netErr) {
		if netErr.Timeout() {
			return protos.AitoolsErrorCategoryTimeout
		}
		return protos.AitoolsErrorCategoryNetwork
	}
	if errors.Is(err, clicompat.ErrNotFound) {
		return protos.AitoolsErrorCategoryManifestNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return protos.AitoolsErrorCategoryPermissionDenied
	}
	// The OS error wrappers: PathError (open, mkdir, ...), LinkError (rename)
	// and SyscallError. Network errors that wrap a SyscallError returned above.
	if isOSError[*fs.PathError](err) || isOSError[*os.LinkError](err) || isOSError[*os.SyscallError](err) {
		return protos.AitoolsErrorCategoryFilesystem
	}
	return protos.AitoolsErrorCategoryUncategorized
}

func isOSError[T error](err error) bool {
	_, ok := errors.AsType[T](err)
	return ok
}

// isErrno reports whether err itself, not something it wraps, is an Errno:
// network errors such as *net.OpError also wrap one.
func isErrno(err error) bool {
	_, ok := err.(syscall.Errno) //nolint:errorlint // must not unwrap, see above
	return ok
}

func skillErrorCategory(e *installer.SkillError) protos.AitoolsErrorCategory {
	switch e.Reason {
	case installer.ReasonSkillNotFound:
		return protos.AitoolsErrorCategorySkillNotFound
	case installer.ReasonVersionIncompatible:
		return protos.AitoolsErrorCategoryVersionIncompatible
	case installer.ReasonExperimentalSkill:
		return protos.AitoolsErrorCategoryExperimentalSkill
	default:
		return protos.AitoolsErrorCategoryUncategorized
	}
}

// fetchErrorCategory maps a FetchError to its category. The first matching
// case wins, so the order matters: an interrupted command is never a fetch
// failure, a transport failure keeps its NETWORK or TIMEOUT category, and the
// release stage is checked before the decode check.
func fetchErrorCategory(e *installer.FetchError) protos.AitoolsErrorCategory {
	// The command was interrupted while the request was in flight: the fetch
	// did not fail on its own.
	if errors.Is(e, context.Canceled) {
		return protos.AitoolsErrorCategoryCommandInterrupted
	}
	// No response arrived (DNS, refused connection, proxy, TLS, client
	// timeout), or reading the body failed on the transport (a timeout or a
	// reset mid-body): the user cannot reach the host, which NETWORK and
	// TIMEOUT already report.
	if c := stdlibErrorCategory(e.Err); e.StatusCode == 0 || c == protos.AitoolsErrorCategoryNetwork || c == protos.AitoolsErrorCategoryTimeout {
		return c
	}
	switch {
	case e.RateLimited:
		return protos.AitoolsErrorCategoryRateLimited
	case errors.Is(e, clicompat.ErrNotFound):
		return protos.AitoolsErrorCategoryManifestNotFound
	// Before the decode check: a release payload that does not decode is a
	// failed release lookup, not an invalid manifest.
	case e.Stage == installer.FetchStageRelease:
		return protos.AitoolsErrorCategoryReleaseLookupFailed
	// The response arrived but was not a manifest: content, not transport.
	case isManifestDecodeError(e) || errors.Is(e, installer.ErrManifestTooLarge):
		return protos.AitoolsErrorCategoryManifestInvalid
	default:
		return protos.AitoolsErrorCategorySkillsFetchFailed
	}
}

// isManifestDecodeError reports whether the response arrived but was not a
// valid manifest.
func isManifestDecodeError(err error) bool {
	_, syntax := errors.AsType[*json.SyntaxError](err)
	_, typ := errors.AsType[*json.UnmarshalTypeError](err)
	return syntax || typ
}

func blockedErrorCategory(e *installer.BlockedError) protos.AitoolsErrorCategory {
	if e.TimedOut {
		return protos.AitoolsErrorCategoryPluginCLITimeout
	}
	switch e.Reason {
	case installer.ReasonCLINotOnPath:
		return protos.AitoolsErrorCategoryCLINotOnPath
	case installer.ReasonInstallFailed:
		return protos.AitoolsErrorCategoryPluginInstallFailed
	default:
		return protos.AitoolsErrorCategoryUncategorized
	}
}
