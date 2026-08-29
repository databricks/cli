package cmdio

import "context"

type limitKeyType struct{}

// WithLimit attaches a result limit to the context.
// Iterator renderers will stop after emitting this many items.
func WithLimit(ctx context.Context, limit int) context.Context {
	return context.WithValue(ctx, limitKeyType{}, limit)
}

// limitFromContext returns the limit from the context, or 0 if none is set.
func limitFromContext(ctx context.Context) int {
	v, ok := ctx.Value(limitKeyType{}).(int)
	if !ok {
		return 0
	}
	return v
}
