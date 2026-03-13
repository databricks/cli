package cmdio

import (
	"context"

	"github.com/databricks/databricks-sdk-go/listing"
)

type limitKey struct{}

// WithLimit stores the limit in the context.
func WithLimit(ctx context.Context, n int) context.Context {
	return context.WithValue(ctx, limitKey{}, n)
}

// GetLimit retrieves the limit from context. Returns 0 if not set.
func GetLimit(ctx context.Context) int {
	v, ok := ctx.Value(limitKey{}).(int)
	if !ok {
		return 0
	}
	return v
}

type limitIterator[T any] struct {
	inner     listing.Iterator[T]
	remaining int
}

func (l *limitIterator[T]) HasNext(ctx context.Context) bool {
	return l.remaining > 0 && l.inner.HasNext(ctx)
}

func (l *limitIterator[T]) Next(ctx context.Context) (T, error) {
	if l.remaining <= 0 {
		var zero T
		return zero, listing.ErrNoMoreItems
	}
	v, err := l.inner.Next(ctx)
	if err != nil {
		return v, err
	}
	l.remaining--
	return v, nil
}

// ApplyLimit wraps a listing.Iterator to yield at most the limit from context.
// It returns the iterator unchanged if the limit is not positive.
func ApplyLimit[T any](ctx context.Context, i listing.Iterator[T]) listing.Iterator[T] {
	if limit := GetLimit(ctx); limit > 0 {
		return &limitIterator[T]{inner: i, remaining: limit}
	}
	return i
}
