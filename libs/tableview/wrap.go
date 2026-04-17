package tableview

import (
	"context"

	"github.com/databricks/databricks-sdk-go/listing"
)

// WrapIterator wraps a typed listing.Iterator into a type-erased RowIterator.
func WrapIterator[T any](iter listing.Iterator[T], columns []ColumnDef) RowIterator {
	return &typedRowIterator[T]{inner: iter, columns: columns}
}

// Col builds a ColumnDef for a typed SDK struct, hiding the type assertion.
func Col[T any](header string, extract func(T) string) ColumnDef {
	return ColumnDef{
		Header:  header,
		Extract: func(v any) string { return extract(v.(T)) },
	}
}

// ColMax is like Col but with a display-width cap.
func ColMax[T any](header string, maxWidth int, extract func(T) string) ColumnDef {
	return ColumnDef{
		Header:   header,
		MaxWidth: maxWidth,
		Extract:  func(v any) string { return extract(v.(T)) },
	}
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
