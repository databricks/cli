//go:build !windows

package agentconfig

// fileInUse reports whether opening a file failed because another process has it open without sharing. Only Windows
// refuses opens that way.
func fileInUse(error) bool {
	return false
}
