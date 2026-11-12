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

// binaryPathFor resolves arg0 without touching the file system beyond lookPath. When neither
// arg0 nor its PATH lookup is absolute, it returns the bare name, which the agent must find on PATH.
func binaryPathFor(arg0 string, lookPath func(string) (string, error)) (string, bool) {
	if filepath.IsAbs(arg0) {
		return arg0, true
	}
	if found, err := lookPath(filepath.Base(arg0)); err == nil && filepath.IsAbs(found) {
		return found, true
	}
	return "databricks", false
}
