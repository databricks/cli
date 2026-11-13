package aitools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/mcp"
)

// binaryArg0 is the name the CLI was started with. It is a variable so that tests can set it.
var binaryArg0 = os.Args[0]

// mcpBinaryPath returns the command that agents run for MCP servers, and whether it is absolute.
// The invoked path is used when it is absolute, so a symlink such as Homebrew's stays the
// stable path; it is never resolved.
func mcpBinaryPath() (string, bool) {
	return binaryPathFor(binaryArg0, exec.LookPath)
}

// binaryPathFor returns the command for arg0, and whether it is absolute. A relative arg0 with a directory is joined
// to the working directory as written, so a symlink followed by ".." resolves as the OS resolves it. A bare name is
// looked up on PATH, and a name that is not found is returned as is, for the agent to find on PATH.
func binaryPathFor(arg0 string, lookPath func(string) (string, error)) (string, bool) {
	if filepath.IsAbs(arg0) {
		return arg0, true
	}
	if filepath.Base(arg0) != arg0 {
		// A leading "./" names the working directory, which the join already gives.
		abs, err := mcp.AbsoluteAsWritten(strings.TrimPrefix(arg0, "."+string(filepath.Separator)))
		if err != nil {
			return arg0, false
		}
		return abs, true
	}
	if found, err := lookPath(arg0); err == nil && filepath.IsAbs(found) {
		return found, true
	}
	return arg0, false
}
