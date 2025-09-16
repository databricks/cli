package cmdctx

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
	// execIdKey is the context key for the execution ID.
	// The value of 1 is arbitrary and can be any number.
	// Other keys in the same package must have different values.
	execIdKey = key(1)

	// configUsedKey is the context key for the auth configuration used to run the
	// command.
	configUsedKey = key(2)

	// workspaceClientKey is the context key for an already configured workspace
	// client that can be used to make authenticated requests.
	workspaceClientKey = key(3)

	// accountClientKey is the context key for an already configured account
	// client that can be used to make authenticated requests.
	accountClientKey = key(4)
)

func GenerateExecId(ctx context.Context) context.Context {
	if v := ctx.Value(execIdKey); v != nil {
		panic("cmdctx.SetExecId called twice on the same context")
	}
	return context.WithValue(ctx, execIdKey, uuid.New().String())
}

// ExecId returns a UUID value that is guaranteed to be the same throughout
// the lifetime of the command execution, and unique for each invocation of the
// CLI.
func ExecId(ctx context.Context) string {
	v := ctx.Value(execIdKey)
	if v == nil {
		panic("cmdctx.ExecId called without calling command.SetExecId first")
	}
	return v.(string)
}
