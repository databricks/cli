package tableview

import (
	"context"

	"github.com/databricks/databricks-sdk-go/listing"
)

// WrapIterator wraps a typed listing.Iterator into a type-erased RowIterator.
func WrapIterator[T any](iter listing.Iterator[T], columns []ColumnDef) RowIterator {
	return &typedRowIterator[T]{inner: iter, columns: columns}
}

type typedRowIterator[T any] struct {
	inner   listing.Iterator[T]
	columns []ColumnDef
}

func (r *typedRowIterator[T]) HasNext(ctx context.Context) bool {
	return r.inner.HasNext(ctx)
}

func (r *typedRowIterator[T]) Next(ctx context.Context) ([]string, error) {
	item, err := r.inner.Next(ctx)
	if err != nil {
		return nil, err
	}
	row := make([]string, len(r.columns))
	for i, col := range r.columns {
		row[i] = col.Extract(item)
	}
	return row, nil
}
