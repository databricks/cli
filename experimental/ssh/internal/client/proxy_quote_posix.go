//go:build !windows

package client

import "strings"

// quoteProxyCommandArgument quotes every argument for the POSIX shell that OpenSSH uses to run ProxyCommand.
func quoteProxyCommandArgument(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
