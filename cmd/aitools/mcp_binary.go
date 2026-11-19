package aitools

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
)

// mcpBinaryPath returns the command that agents run for MCP servers, and whether it is absolute.
// The invoked path is used when it is absolute, so a symlink such as Homebrew's stays the
// stable path; it is never resolved. An empty argv has no invoked path, so the caller refuses
// the empty command as not being the databricks binary.
func mcpBinaryPath() (string, bool) {
	if len(os.Args) == 0 {
		return "", false
	}
	return binaryPathFor(os.Args[0], exec.LookPath, os.Executable)
}

// binaryPathFor returns the command for arg0, and whether it is absolute. A relative arg0 with a directory is joined
// to the working directory as written, so a symlink followed by ".." resolves as the OS resolves it. A bare name is
// looked up on PATH. A binary found only in the working directory is the running executable instead, and a name that is
// not found is returned as is, for the agent to find on PATH.
func binaryPathFor(arg0 string, lookPath func(string) (string, error), executable func() (string, error)) (string, bool) {
	if filepath.IsAbs(arg0) {
		return arg0, true
	}
	if filepath.Base(arg0) != arg0 {
		// A leading "./" names the working directory, which the join already gives. The separators after it are not a
		// root, so they are dropped with it: ".//bin" stays relative.
		rel := filepath.FromSlash(arg0)
		if after, ok := strings.CutPrefix(rel, "."+string(filepath.Separator)); ok {
			rel = strings.TrimLeft(after, string(filepath.Separator))
		}
		abs, err := mcpconfig.AbsoluteAsWritten(rel)
		if err != nil {
			return arg0, false
		}
		return abs, true
	}
	found, err := lookPath(arg0)
	if err == nil && filepath.IsAbs(found) {
		return found, true
	}
	// On Windows the lookup also searches the working directory, and LookPath returns exec.ErrDot for a binary found
	// only there. The running executable is that binary, and its path is absolute.
	if errors.Is(err, exec.ErrDot) {
		if running, runErr := executable(); runErr == nil {
			return running, true
		}
	}
	return arg0, false
}
