package tableview

import "context"

// ColumnDef defines a column in the TUI table.
type ColumnDef struct {
	Header string // Display name in header row.
	// MaxWidth caps cell display width; 0 = default (50). Values exceeding
	// this limit are destructively truncated with "..." in the rendered
	// output. Horizontal scrolling does not recover the hidden portion.
	MaxWidth int
	Extract  func(v any) string // Extracts cell value from typed SDK struct.
}

// SearchConfig configures server-side search for a list command.
type SearchConfig struct {
	Placeholder string // Shown in search bar.
	// NewIterator creates a fresh RowIterator with the search applied.
	// Called when user submits a search query.
	NewIterator func(ctx context.Context, query string) RowIterator
}

// TableConfig configures the TUI table for a list command.
type TableConfig struct {
	Columns []ColumnDef
	Search  *SearchConfig // nil = search disabled.
}

// RowIterator provides type-erased rows to the TUI.
type RowIterator interface {
	HasNext(ctx context.Context) bool
	Next(ctx context.Context) ([]string, error)
}
