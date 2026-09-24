//go:build windows

package client_test

import "syscall"

func quoteProxyCommandArgumentForTest(arg string) string {
	return syscall.EscapeArg(arg)
}
