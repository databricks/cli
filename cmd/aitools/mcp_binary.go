package aitools

import (
	"os"
	"os/exec"
	"path/filepath"
)

// mcpBinaryPath returns the command that agents run for MCP servers, and whether it is absolute.
// The invoked path is used when it is absolute, so a symlink such as Homebrew's stays the
// stable path; it is never resolved.
func mcpBinaryPath() (string, bool) {
	return binaryPathFor(os.Args[0], exec.LookPath)
}

// binaryPathFor resolves arg0 without touching the file system beyond lookPath. An absolute arg0 is kept.
// A relative arg0 with a directory, such as ./databricks, is made absolute, and its symlinks are not
// resolved. A bare name is looked up on PATH. When none of these gives an absolute path, it returns arg0
// as the bare name, which the agent must find on PATH. The caller checks that the name is databricks.
func binaryPathFor(arg0 string, lookPath func(string) (string, error)) (string, bool) {
	if filepath.IsAbs(arg0) {
		return arg0, true
	}
	if filepath.Base(arg0) != arg0 {
		abs, err := filepath.Abs(arg0)
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
