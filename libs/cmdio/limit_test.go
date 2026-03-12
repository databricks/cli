package cmdio_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sliceIterator[T any] struct {
	items []T
}

func (s *sliceIterator[T]) HasNext(_ context.Context) bool {
	return len(s.items) > 0
}

func (s *sliceIterator[T]) Next(_ context.Context) (T, error) {
	if len(s.items) == 0 {
		var zero T
		return zero, errors.New("no more items")
	}
	item := s.items[0]
	s.items = s.items[1:]
	return item, nil
}

func drain[T any](ctx context.Context, iter listing.Iterator[T]) ([]T, error) {
	var result []T
	for iter.HasNext(ctx) {
		v, err := iter.Next(ctx)
		if err != nil {
			return result, err
		}
		result = append(result, v)
	}
	return result, nil
}

type errorIterator[T any] struct {
	items     []T
	failAt    int
	callCount int
}

func (e *errorIterator[T]) HasNext(_ context.Context) bool {
	return e.callCount <= e.failAt && e.callCount < len(e.items)
}

func (e *errorIterator[T]) Next(_ context.Context) (T, error) {
	idx := e.callCount
	e.callCount++
	if idx == e.failAt {
		var zero T
		return zero, errors.New("fetch error")
	}
	return e.items[idx], nil
}

func TestWithLimitRoundTrip(t *testing.T) {
	ctx := cmdio.WithLimit(t.Context(), 42)
	assert.Equal(t, 42, cmdio.GetLimit(ctx))
}

func TestGetLimitReturnsZeroWhenNotSet(t *testing.T) {
	assert.Equal(t, 0, cmdio.GetLimit(t.Context()))
}

func TestApplyLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		setLimit bool
		items    []int
		want     []int
	}{
		{
			name:     "caps results",
			limit:    5,
			setLimit: true,
			items:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			want:     []int{1, 2, 3, 4, 5},
		},
		{
			name:  "no-op when unset",
			items: []int{1, 2, 3},
			want:  []int{1, 2, 3},
		},
		{
			name:     "greater than total",
			limit:    10,
			setLimit: true,
			items:    []int{1, 2, 3},
			want:     []int{1, 2, 3},
		},
		{
			name:     "one",
			limit:    1,
			setLimit: true,
			items:    []int{1, 2, 3},
			want:     []int{1},
		},
		{
			name:     "zero",
			limit:    0,
			setLimit: true,
			items:    []int{1, 2, 3},
			want:     []int{1, 2, 3},
		},
		{
			name:     "negative",
			limit:    -1,
			setLimit: true,
			items:    []int{1, 2, 3},
			want:     []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			if tt.setLimit {
				ctx = cmdio.WithLimit(ctx, tt.limit)
			}

			iter := cmdio.ApplyLimit(ctx, &sliceIterator[int]{items: tt.items})

			result, err := drain(t.Context(), iter)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestApplyLimitPreservesErrors(t *testing.T) {
	ctx := cmdio.WithLimit(t.Context(), 5)
	iter := cmdio.ApplyLimit(ctx, &errorIterator[int]{items: []int{1, 2, 3}, failAt: 2})

	result, err := drain(t.Context(), iter)
	assert.ErrorContains(t, err, "fetch error")
	assert.Equal(t, []int{1, 2}, result)
}
