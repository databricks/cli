//go:build windows

package autoinstall

import "context"

// RootInUserHome reports whether the CLI runs as root in a home directory
// that another user owns. Windows has no root user that the CLI switches to:
// an elevated process keeps the user's own profile, so it reports false.
func RootInUserHome(context.Context) bool {
	return false
}
