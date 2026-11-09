package aitools

import (
	"context"
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

// classifyInstallError maps an aitools error to its telemetry category without
// reading any error text. Typed installer errors win, then standard library
// error types.
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
	return stdlibErrorCategory(err)
}

// stdlibErrorCategory classifies errors by their standard library type, which
// the installer wraps with %w: network and timeout errors from fetching the
// manifest or skill files, and filesystem errors from writing skills or the
// install state. JSON decode errors stay UNCATEGORIZED: the manifest, the
// release payload and the install state all decode JSON, so only typed
// installer errors can tell MANIFEST_INVALID from STATE_INVALID.
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

func blockedErrorCategory(e *installer.BlockedError) protos.AitoolsErrorCategory {
	switch e.Reason {
	case installer.ReasonCLINotOnPath:
		return protos.AitoolsErrorCategoryCLINotOnPath
	case installer.ReasonInstallFailed:
		return protos.AitoolsErrorCategoryPluginInstallFailed
	default:
		return protos.AitoolsErrorCategoryUncategorized
	}
}
