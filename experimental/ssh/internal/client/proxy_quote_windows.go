//go:build windows

package client

import "syscall"

// quoteProxyCommandArgument quotes every argument for the Windows command-line parser used by OpenSSH.
func quoteProxyCommandArgument(arg string) string {
	return syscall.EscapeArg(arg)
}
