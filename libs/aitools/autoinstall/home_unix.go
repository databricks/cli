//go:build unix

package autoinstall

import (
	"context"
	"os"

	"github.com/databricks/cli/libs/env"
	"golang.org/x/sys/unix"
)

// RootInUserHome reports whether the CLI runs as root in a home directory
// that another user owns, as under sudo with HOME kept (the macOS default,
// and `sudo -E` on Linux). What auto-install writes there, its state and the
// agents' files, would belong to root, and the user could no longer change
// it. A home that cannot be inspected counts as not. Only the home is
// checked: an agent directory set through an environment variable, such as
// CLAUDE_CONFIG_DIR kept by `sudo -E -H`, is out of scope.
func RootInUserHome(ctx context.Context) bool {
	return rootInUserHome(ctx, os.Geteuid())
}

// rootInUserHome is RootInUserHome for the effective user ID euid.
func rootInUserHome(ctx context.Context, euid int) bool {
	if euid != 0 {
		return false
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	var st unix.Stat_t
	if err := unix.Stat(home, &st); err != nil {
		return false
	}
	return st.Uid != 0
}
