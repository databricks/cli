package cmdio

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/stretchr/testify/assert"
)

type dummyIterator struct {
	items []any
}

func (d *dummyIterator) HasNext(_ context.Context) bool {
	return len(d.items) > 0
}

func (d *dummyIterator) Next(ctx context.Context) (any, error) {
	if !d.HasNext(ctx) {
		return nil, errors.New("no more items")
	}
	item := d.items[0]
	d.items = d.items[1:]
	return item, nil
}

var _ listing.Iterator[any] = &dummyIterator{}

func TestReflectIterator_NonIterator(t *testing.T) {
	_, ok := newReflectIterator(3)
	assert.False(t, ok)
}

func TestReflectIterator_DummyIterator(t *testing.T) {
	ri, ok := newReflectIterator(&dummyIterator{items: []any{1, "2", true}})
	assert.True(t, ok)
	ctx := context.Background()
	first, err := ri.Next(ctx)
	assert.NoError(t, err)
	assert.Equal(t, first, 1)
	second, err := ri.Next(ctx)
	assert.NoError(t, err)
	assert.Equal(t, second, "2")
	third, err := ri.Next(ctx)
	assert.NoError(t, err)
	assert.Equal(t, third, true)
	assert.False(t, ri.HasNext(ctx))
}
