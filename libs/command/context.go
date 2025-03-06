package command

import (
	"context"

	"github.com/google/uuid"
)

// key is a package-local type to use for context keys.
//
// Using an unexported type for context keys prevents key collisions across
// packages since external packages cannot create values of this type.
type key int

const (
	// execIDKey is the context key for the execution ID.
	// The value of 1 is arbitrary and can be any number.
	// Other keys in the same package must have different values.
	execIDKey = key(1)
)

func SetExecId(ctx context.Context) context.Context {
	if v := ctx.Value(execIDKey); v != nil {
		panic("command.SetExecId called twice on the same context")
	}
	return context.WithValue(ctx, execIDKey, uuid.New().String())
}

func MockExecId(ctx context.Context, v string) context.Context {
	if v := ctx.Value(execIDKey); v != nil {
		panic("command.MockExecId called twice on the same context")
	}
	return context.WithValue(ctx, execIDKey, v)
}

// ExecId returns a UUID value that is guaranteed to be the same throughout
// the lifetime of the command execution, and unique for each invocation of the
// CLI.
func ExecId(ctx context.Context) string {
	v := ctx.Value(execIDKey)
	if v == nil {
		panic("command.ExecId called without calling command.SetExecId first")
	}
	return v.(string)
}
