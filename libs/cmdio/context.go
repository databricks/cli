package cmdio

import (
	"context"

	"github.com/spf13/cobra"
)

type cmdKeyType struct{}

// WithCommand stores the cobra.Command in context.
func WithCommand(ctx context.Context, cmd *cobra.Command) context.Context {
	return context.WithValue(ctx, cmdKeyType{}, cmd)
}

// CommandFromContext retrieves the cobra.Command from context.
func CommandFromContext(ctx context.Context) *cobra.Command {
	cmd, _ := ctx.Value(cmdKeyType{}).(*cobra.Command)
	return cmd
}

type maxItemsKeyType struct{}

// WithMaxItems stores a max items limit in context.
func WithMaxItems(ctx context.Context, n int) context.Context {
	return context.WithValue(ctx, maxItemsKeyType{}, n)
}

// GetMaxItems retrieves the max items limit from context (0 = unlimited).
func GetMaxItems(ctx context.Context) int {
	n, _ := ctx.Value(maxItemsKeyType{}).(int)
	return n
}
