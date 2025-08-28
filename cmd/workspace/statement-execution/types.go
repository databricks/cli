package statement_execution

import (
	"time"
)

// ExecuteStatementRequest represents the request body for executing a SQL statement
type ExecuteStatementRequest struct {
	Statement     string               `json:"statement"`
	WarehouseId   string               `json:"warehouse_id"`
	Catalog       string               `json:"catalog,omitempty"`
	Schema        string               `json:"schema,omitempty"`
	Disposition   string               `json:"disposition,omitempty"`
	Format        string               `json:"format,omitempty"`
	WaitTimeout   string               `json:"wait_timeout,omitempty"`
	OnWaitTimeout string               `json:"on_wait_timeout,omitempty"`
	RowLimit      int64                `json:"row_limit,omitempty"`
	ByteLimit     int64                `json:"byte_limit,omitempty"`
	Parameters    []StatementParameter `json:"parameters,omitempty"`
}

// StatementParameter represents a parameter for a parameterized SQL statement
type StatementParameter struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
}

// StatementResponse represents the response from executing a SQL statement
type StatementResponse struct {
	StatementId   string          `json:"statement_id"`
	Status        StatementStatus `json:"status"`
	Manifest      *ResultManifest `json:"manifest,omitempty"`
	Result        *InlineResult   `json:"result,omitempty"`
	ExternalLinks []ExternalLink  `json:"external_links,omitempty"`
}

// StatementStatus represents the execution status of a statement
type StatementStatus struct {
	State string          `json:"state"`
	Error *StatementError `json:"error,omitempty"`
}

// StatementError represents an error that occurred during statement execution
type StatementError struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// ResultManifest provides schema and metadata for the result set
type ResultManifest struct {
	Schema    ResultSchema `json:"schema"`
	TotalRows int64        `json:"total_rows,omitempty"`
	Truncated bool         `json:"truncated,omitempty"`
}

// ResultSchema describes the schema of the result set
type ResultSchema struct {
	Columns []ColumnInfo `json:"columns"`
}

// ColumnInfo describes a column in the result set
type ColumnInfo struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	TypeName      string `json:"type_name,omitempty"`
	Position      int    `json:"position,omitempty"`
	TypePrecision int    `json:"type_precision,omitempty"`
	TypeScale     int    `json:"type_scale,omitempty"`
}

// InlineResult contains the result data when using INLINE disposition
type InlineResult struct {
	DataArray [][]any `json:"data_array"`
}

// ExternalLink represents a link to external result data
type ExternalLink struct {
	URL        string    `json:"url"`
	Expiration time.Time `json:"expiration"`
	ByteSize   int64     `json:"byte_size,omitempty"`
	RowCount   int64     `json:"row_count,omitempty"`
}
