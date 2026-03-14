package tableview

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeItem struct {
	Name string
	Age  int
}

type fakeIterator[T any] struct {
	items []T
	pos   int
}

func (f *fakeIterator[T]) HasNext(_ context.Context) bool {
	return f.pos < len(f.items)
}

func (f *fakeIterator[T]) Next(_ context.Context) (T, error) {
	if f.pos >= len(f.items) {
		var zero T
		return zero, errors.New("no more items")
	}
	item := f.items[f.pos]
	f.pos++
	return item, nil
}

func TestWrapIteratorNormalIteration(t *testing.T) {
	items := []fakeItem{{Name: "alice", Age: 30}, {Name: "bob", Age: 25}}
	iter := &fakeIterator[fakeItem]{items: items}
	columns := []ColumnDef{
		{Header: "Name", Extract: func(v any) string { return v.(fakeItem).Name }},
		{Header: "Age", Extract: func(v any) string { return strconv.Itoa(v.(fakeItem).Age) }},
	}

	ctx := t.Context()
	ri := WrapIterator[fakeItem](iter, columns)

	require.True(t, ri.HasNext(ctx))
	row, err := ri.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "30"}, row)

	require.True(t, ri.HasNext(ctx))
	row, err = ri.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"bob", "25"}, row)

	assert.False(t, ri.HasNext(ctx))
}

func TestWrapIteratorEmpty(t *testing.T) {
	iter := &fakeIterator[fakeItem]{}
	columns := []ColumnDef{
		{Header: "Name", Extract: func(v any) string { return v.(fakeItem).Name }},
	}

	ctx := t.Context()
	ri := WrapIterator[fakeItem](iter, columns)
	assert.False(t, ri.HasNext(ctx))
}

func TestWrapIteratorExtractFunctions(t *testing.T) {
	items := []fakeItem{{Name: "charlie", Age: 42}}
	iter := &fakeIterator[fakeItem]{items: items}
	columns := []ColumnDef{
		{Header: "Upper", Extract: func(v any) string { return "PREFIX_" + v.(fakeItem).Name }},
	}

	ctx := t.Context()
	ri := WrapIterator[fakeItem](iter, columns)
	row, err := ri.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"PREFIX_charlie"}, row)
}
