# Phase 2: Databricks Integration Improvements

**Priority**: High
**Dependencies**: Phase 1 (for environment validation)
**Estimated Effort**: Medium

## Overview

This phase enhances the Databricks provider with pagination support and improved data retrieval for handling large catalogs and datasets efficiently.

---

## 2.1 Pagination Support

**Rust Reference**: PR #559 (commit `66473459`)

### Description

Add pagination capabilities to the `databricks_list_tables` tool to handle large catalogs with many tables. This prevents timeouts and improves performance when working with extensive data catalogs.

### Changes Required

#### Modified Files

1. **`experimental/apps-mcp/lib/providers/databricks/tables.go`**
   - Update `ListTablesInput` struct:
     ```go
     type ListTablesInput struct {
         Catalog    string  `json:"catalog" jsonschema:"required,description=Catalog name"`
         Schema     string  `json:"schema" jsonschema:"required,description=Schema name"`
         PageSize   *int    `json:"page_size,omitempty" jsonschema:"description=Number of tables to return (default: 100, max: 1000)"`
         PageToken  *string `json:"page_token,omitempty" jsonschema:"description=Token for next page of results"`
         Filter     *string `json:"filter,omitempty" jsonschema:"description=Optional filter pattern for table names (supports wildcards)"`
     }
     ```
   - Update `ListTablesOutput` struct:
     ```go
     type ListTablesOutput struct {
         Tables        []TableInfo `json:"tables"`
         NextPageToken *string     `json:"next_page_token,omitempty"`
         TotalCount    int         `json:"total_count"`
     }
     ```
   - Implement pagination logic in `ListTables()` function
   - Add filtering logic for table names

2. **`experimental/apps-mcp/lib/providers/databricks/provider.go`**
   - Update tool schema to reflect new parameters
   - Update documentation/description for pagination

3. **`experimental/apps-mcp/lib/providers/databricks/format.go`**
   - Update formatting functions to handle paginated results
   - Add indication when more results are available

### Implementation Notes

```go
func (p *Provider) ListTables(ctx context.Context, input ListTablesInput) (*ListTablesOutput, error) {
    pageSize := 100
    if input.PageSize != nil {
        pageSize = min(*input.PageSize, 1000)
    }

    // Use Databricks SDK with pagination
    listTablesRequest := catalog.ListTablesRequest{
        CatalogName:   input.Catalog,
        SchemaName:    input.Schema,
        MaxResults:    pageSize,
    }

    if input.PageToken != nil {
        listTablesRequest.PageToken = *input.PageToken
    }

    iterator := p.workspace.Tables.List(ctx, listTablesRequest)

    var tables []TableInfo
    var nextToken *string

    for iterator.HasNext(ctx) {
        table, err := iterator.Next(ctx)
        if err != nil {
            return nil, err
        }

        // Apply filter if provided
        if input.Filter != nil && !matchFilter(table.Name, *input.Filter) {
            continue
        }

        tables = append(tables, convertTableInfo(table))

        if len(tables) >= pageSize {
            if token := iterator.PageToken(); token != "" {
                nextToken = &token
            }
            break
        }
    }

    return &ListTablesOutput{
        Tables:        tables,
        NextPageToken: nextToken,
        TotalCount:    len(tables),
    }, nil
}
```

### Testing

- Unit tests for pagination logic
  - First page retrieval
  - Subsequent pages with page tokens
  - Last page (no next token)
  - Page size limits
- Filter pattern matching tests
  - Exact matches
  - Wildcard patterns
  - Case sensitivity
- Integration tests with real Databricks workspace
  - Large catalogs (>1000 tables)
  - Empty schemas
  - Invalid page tokens

### Success Criteria

- [x] `databricks_list_tables` supports `page_size` parameter
- [x] `databricks_list_tables` supports `page_token` parameter
- [x] `databricks_list_tables` supports `filter` parameter
- [x] Returns `next_page_token` when more results available
- [x] Works with existing code (backward compatible)
- [x] Handles edge cases (empty results, last page, etc.)

---

## 2.2 Enhanced Table Operations

**Rust Reference**: PR #559 (commit `66473459`)

### Description

Improve data retrieval efficiency for large result sets and add better error handling for Databricks SQL operations.

### Changes Required

#### Modified Files

1. **`experimental/apps-mcp/lib/providers/databricks/sql.go`**
   - Add result size limiting for `databricks_execute_query`
   - Add streaming support for large result sets
   - Implement better timeout handling
   - Add progress indicators for long-running queries
   - Update `ExecuteQueryInput` struct:
     ```go
     type ExecuteQueryInput struct {
         Query        string  `json:"query" jsonschema:"required,description=SQL query to execute"`
         WarehouseId  *string `json:"warehouse_id,omitempty" jsonschema:"description=SQL warehouse ID (uses default from config if not provided)"`
         MaxRows      *int    `json:"max_rows,omitempty" jsonschema:"description=Maximum rows to return (default: 1000, max: 10000)"`
         Timeout      *int    `json:"timeout,omitempty" jsonschema:"description=Query timeout in seconds (default: 60)"`
     }
     ```
   - Update `ExecuteQueryOutput` struct:
     ```go
     type ExecuteQueryOutput struct {
         Columns      []string        `json:"columns"`
         Rows         [][]interface{} `json:"rows"`
         RowCount     int             `json:"row_count"`
         Truncated    bool            `json:"truncated"`
         ExecutionTime float64        `json:"execution_time_seconds"`
     }
     ```

2. **`experimental/apps-mcp/lib/providers/databricks/format.go`**
   - Add formatting for large result sets
   - Add truncation indicators
   - Improve table formatting for better readability

### Implementation Notes

```go
func (p *Provider) ExecuteQuery(ctx context.Context, input ExecuteQueryInput) (*ExecuteQueryOutput, error) {
    maxRows := 1000
    if input.MaxRows != nil {
        maxRows = min(*input.MaxRows, 10000)
    }

    timeout := 60 * time.Second
    if input.Timeout != nil {
        timeout = time.Duration(*input.Timeout) * time.Second
    }

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    startTime := time.Now()

    // Execute query with SQL execution API
    result, err := p.executeWithRetry(ctx, input.Query, warehouseId)
    if err != nil {
        return nil, fmt.Errorf("query execution failed: %w", err)
    }

    // Stream results and limit rows
    rows, truncated := p.streamResults(result, maxRows)
    executionTime := time.Since(startTime).Seconds()

    return &ExecuteQueryOutput{
        Columns:       result.Columns,
        Rows:          rows,
        RowCount:      len(rows),
        Truncated:     truncated,
        ExecutionTime: executionTime,
    }, nil
}

func (p *Provider) executeWithRetry(ctx context.Context, query string, warehouseId string) (*sql.Result, error) {
    // Implement retry logic with exponential backoff
    // Handle warehouse cold start scenarios
    // Better error messages for common issues
}
```

### Testing

- Query execution tests
  - Small result sets
  - Large result sets (>1000 rows)
  - Result truncation
  - Timeout handling
- Error handling tests
  - Invalid SQL syntax
  - Missing warehouse
  - Connection failures
  - Query timeouts
- Performance tests
  - Query execution time tracking
  - Streaming performance
  - Memory usage with large results

### Success Criteria

- [x] `databricks_execute_query` supports `max_rows` parameter
- [x] `databricks_execute_query` supports `timeout` parameter
- [x] `databricks_execute_query` supports `warehouse_id` parameter
- [x] Returns execution time in output
- [x] Indicates when results are truncated
- [x] Better error messages for common failures
- [x] Memory-efficient for large result sets

---

## Testing Strategy

1. **Unit Tests**: Mock Databricks SDK responses
2. **Integration Tests**: Use real Databricks workspace with test data
3. **Performance Tests**: Measure query execution time and memory usage
4. **Load Tests**: Test with large catalogs and result sets

## Migration Guide

For users of the existing API:

### Before (without pagination)
```json
{
  "catalog": "main",
  "schema": "default"
}
```

### After (with pagination)
```json
{
  "catalog": "main",
  "schema": "default",
  "page_size": 100,
  "page_token": "optional_token_from_previous_response"
}
```

**Note**: Existing calls will continue to work (backward compatible) with default page size of 100.

## Dependencies

- Phase 1.1 (Environment Management) for warehouse ID validation
- Databricks Go SDK version check (may need update for pagination APIs)

## Related Files in Rust Implementation

- `edda/edda_integrations/src/databricks.rs` - Pagination implementation
- `edda/edda_agent/src/processor/databricks.rs` - Query execution improvements
- `edda/edda_mcp/src/providers/databricks.rs` - Provider updates
