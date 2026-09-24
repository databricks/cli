//go:build !windows

package client_test

import "strings"

func quoteProxyCommandArgumentForTest(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
