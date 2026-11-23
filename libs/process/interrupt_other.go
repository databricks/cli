//go:build !unix

package process

import (
	"context"
	"os"
)

// BackgroundInGroup runs args as Background does, with WithProcessGroup. On
// these platforms the child stays in the console's process group, so Ctrl-C
// reaches it directly and nothing is forwarded.
func BackgroundInGroup(ctx context.Context, args []string, opts ...execOption) (string, error) {
	return Background(ctx, args, append(opts, WithProcessGroup())...)
}

// Raise does nothing on these platforms, where a process cannot end itself by
// a signal: the caller carries on and exits as it would after any other
// cancellation.
func Raise(os.Signal) {}
