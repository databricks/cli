//go:build !unix

package process

import "context"

// BackgroundInGroup runs args as Background does, with WithProcessGroup. On
// these platforms the child stays in the console's process group, so Ctrl-C
// reaches it directly and nothing is forwarded.
func BackgroundInGroup(ctx context.Context, args []string, opts ...execOption) (string, error) {
	return Background(ctx, args, append(opts, WithProcessGroup())...)
}
