# Step 4: Databricks Provider

## Overview
Implement the Databricks provider using the official Databricks SDK for Go, exposing tools for catalog exploration, table inspection, and SQL query execution.

## Tasks

### 4.1 Add Databricks SDK Dependency

```bash
go get github.com/databricks/databricks-sdk-go
```

Study the SDK documentation:
- https://docs.databricks.com/dev-tools/sdk-go.html
- https://pkg.go.dev/github.com/databricks/databricks-sdk-go

### 4.2 Create Databricks Client

**pkg/providers/databricks/client.go:**

```go
type Client struct {
    workspace *databricks.WorkspaceClient
    config    *config.Config
    logger    *slog.Logger
}

func NewClient(cfg *config.Config, logger *slog.Logger) (*Client, error) {
    // Create workspace client from environment variables
    // DATABRICKS_HOST, DATABRICKS_TOKEN
    workspace, err := databricks.NewWorkspaceClient()
    if err != nil {
        return nil, fmt.Errorf("failed to create Databricks client: %w", err)
    }

    return &Client{
        workspace: workspace,
        config:    cfg,
        logger:    logger,
    }, nil
}
```

### 4.3 Implement Catalog Operations

**pkg/providers/databricks/catalogs.go:**

```go
type ListCatalogsResult struct {
    Catalogs []string `json:"catalogs"`
}

func (c *Client) ListCatalogs(ctx context.Context) (*ListCatalogsResult, error) {
    catalogs, err := c.workspace.Catalogs.ListAll(ctx, catalog.ListCatalogsRequest{})
    if err != nil {
        return nil, fmt.Errorf("failed to list catalogs: %w", err)
    }

    names := make([]string, len(catalogs))
    for i, cat := range catalogs {
        names[i] = cat.Name
    }

    return &ListCatalogsResult{Catalogs: names}, nil
}

type ListSchemasArgs struct {
    CatalogName string `json:"catalog_name"`
    Filter      string `json:"filter,omitempty"`
    Limit       int    `json:"limit,omitempty"`
    Offset      int    `json:"offset,omitempty"`
}

type ListSchemasResult struct {
    Schemas    []string `json:"schemas"`
    TotalCount int      `json:"total_count"`
    ShownCount int      `json:"shown_count"`
    Offset     int      `json:"offset"`
    Limit      int      `json:"limit"`
}

func (c *Client) ListSchemas(ctx context.Context, args *ListSchemasArgs) (*ListSchemasResult, error) {
    if args.Limit == 0 {
        args.Limit = 1000
    }

    schemas, err := c.workspace.Schemas.ListAll(ctx, catalog.ListSchemasRequest{
        CatalogName: args.CatalogName,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to list schemas: %w", err)
    }

    // Extract names
    names := make([]string, len(schemas))
    for i, schema := range schemas {
        names[i] = schema.Name
    }

    // Apply filter if provided
    if args.Filter != "" {
        filtered := make([]string, 0)
        filterLower := strings.ToLower(args.Filter)
        for _, name := range names {
            if strings.Contains(strings.ToLower(name), filterLower) {
                filtered = append(filtered, name)
            }
        }
        names = filtered
    }

    // Apply pagination
    totalCount := len(names)
    start := args.Offset
    end := start + args.Limit

    if start > len(names) {
        start = len(names)
    }
    if end > len(names) {
        end = len(names)
    }

    paginatedNames := names[start:end]

    return &ListSchemasResult{
        Schemas:    paginatedNames,
        TotalCount: totalCount,
        ShownCount: len(paginatedNames),
        Offset:     args.Offset,
        Limit:      args.Limit,
    }, nil
}

type ListTablesArgs struct {
    CatalogName         string `json:"catalog_name"`
    SchemaName          string `json:"schema_name"`
    ExcludeInaccessible bool   `json:"exclude_inaccessible"`
}

type TableInfo struct {
    Name        string  `json:"name"`
    CatalogName string  `json:"catalog_name"`
    SchemaName  string  `json:"schema_name"`
    FullName    string  `json:"full_name"`
    TableType   string  `json:"table_type"`
    Owner       *string `json:"owner,omitempty"`
    Comment     *string `json:"comment,omitempty"`
}

type ListTablesResult struct {
    Tables []TableInfo `json:"tables"`
}

func (c *Client) ListTables(ctx context.Context, args *ListTablesArgs) (*ListTablesResult, error) {
    tables, err := c.workspace.Tables.ListAll(ctx, catalog.ListTablesRequest{
        CatalogName:     args.CatalogName,
        SchemaName:      args.SchemaName,
        IncludeBrowse:   !args.ExcludeInaccessible,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to list tables: %w", err)
    }

    infos := make([]TableInfo, len(tables))
    for i, table := range tables {
        infos[i] = TableInfo{
            Name:        table.Name,
            CatalogName: table.CatalogName,
            SchemaName:  table.SchemaName,
            FullName:    fmt.Sprintf("%s.%s.%s", table.CatalogName, table.SchemaName, table.Name),
            TableType:   string(table.TableType),
            Owner:       table.Owner,
            Comment:     table.Comment,
        }
    }

    return &ListTablesResult{Tables: infos}, nil
}
```

### 4.4 Implement Table Description

**pkg/providers/databricks/tables.go:**

```go
type DescribeTableArgs struct {
    TableFullName string `json:"table_full_name"`
    SampleSize    int    `json:"sample_size,omitempty"`
}

type ColumnMetadata struct {
    Name     string  `json:"name"`
    DataType string  `json:"data_type"`
    Comment  *string `json:"comment,omitempty"`
}

type TableDetails struct {
    FullName         string           `json:"full_name"`
    TableType        string           `json:"table_type"`
    Owner            *string          `json:"owner,omitempty"`
    Comment          *string          `json:"comment,omitempty"`
    StorageLocation  *string          `json:"storage_location,omitempty"`
    DataSourceFormat *string          `json:"data_source_format,omitempty"`
    Columns          []ColumnMetadata `json:"columns"`
    SampleData       []map[string]any `json:"sample_data,omitempty"`
    RowCount         *int64           `json:"row_count,omitempty"`
}

func (c *Client) DescribeTable(ctx context.Context, args *DescribeTableArgs) (*TableDetails, error) {
    if args.SampleSize == 0 {
        args.SampleSize = 5
    }

    // Get table metadata
    tableInfo, err := c.workspace.Tables.Get(ctx, catalog.GetTableRequest{
        FullName: args.TableFullName,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get table: %w", err)
    }

    // Build column metadata
    columns := make([]ColumnMetadata, len(tableInfo.Columns))
    for i, col := range tableInfo.Columns {
        columns[i] = ColumnMetadata{
            Name:     col.Name,
            DataType: string(col.TypeName),
            Comment:  col.Comment,
        }
    }

    details := &TableDetails{
        FullName:         args.TableFullName,
        TableType:        string(tableInfo.TableType),
        Owner:            tableInfo.Owner,
        Comment:          tableInfo.Comment,
        StorageLocation:  tableInfo.StorageLocation,
        DataSourceFormat: (*string)(tableInfo.DataSourceFormat),
        Columns:          columns,
    }

    // Get sample data if requested
    if args.SampleSize > 0 {
        query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", args.TableFullName, args.SampleSize)
        sampleData, err := c.ExecuteQuery(ctx, query)
        if err == nil && len(sampleData) > 0 {
            details.SampleData = sampleData
        }
    }

    // Get row count
    countQuery := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", args.TableFullName)
    countData, err := c.ExecuteQuery(ctx, countQuery)
    if err == nil && len(countData) > 0 {
        if count, ok := countData[0]["count"].(int64); ok {
            details.RowCount = &count
        }
    }

    return details, nil
}
```

### 4.5 Implement SQL Execution

**pkg/providers/databricks/sql.go:**

```go
type ExecuteQueryArgs struct {
    Query string `json:"query"`
}

func (c *Client) ExecuteQuery(ctx context.Context, query string) ([]map[string]any, error) {
    // Get warehouse ID from config
    if c.config.WarehouseID == "" {
        return nil, errors.New("DATABRICKS_WAREHOUSE_ID not configured")
    }

    // Execute statement
    result, err := c.workspace.StatementExecution.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
        Statement:   query,
        WarehouseId: c.config.WarehouseID,
        WaitTimeout: "30s",
        Format:      sql.FormatJsonArray,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to execute query: %w", err)
    }

    // Check status
    if result.Status.State == sql.StatementStateFailed {
        errMsg := "unknown error"
        if result.Status.Error != nil && result.Status.Error.Message != "" {
            errMsg = result.Status.Error.Message
        }
        return nil, fmt.Errorf("query failed: %s", errMsg)
    }

    // Parse results
    if result.Result == nil || result.Result.DataArray == nil {
        return []map[string]any{}, nil
    }

    // Get column names
    columns := make([]string, len(result.Manifest.Schema.Columns))
    for i, col := range result.Manifest.Schema.Columns {
        columns[i] = col.Name
    }

    // Convert data array to map
    rows := make([]map[string]any, len(result.Result.DataArray))
    for i, row := range result.Result.DataArray {
        rowMap := make(map[string]any)
        for j, val := range row {
            if j < len(columns) {
                rowMap[columns[j]] = val
            }
        }
        rows[i] = rowMap
    }

    return rows, nil
}
```

### 4.6 Create Provider Implementation

**pkg/providers/databricks/provider.go:**

```go
type Provider struct {
    client   *Client
    registry *mcp.Registry
    logger   *slog.Logger
}

func NewProvider(cfg *config.Config, logger *slog.Logger) (*Provider, error) {
    client, err := NewClient(cfg, logger)
    if err != nil {
        return nil, err
    }

    p := &Provider{
        client:   client,
        registry: mcp.NewRegistry(),
        logger:   logger,
    }

    // Register tools
    if err := p.registerTools(); err != nil {
        return nil, err
    }

    return p, nil
}

func (p *Provider) registerTools() error {
    tools := []struct {
        tool mcp.Tool
        fn   mcp.ToolFunc
    }{
        {
            tool: mcp.Tool{
                Name:        "databricks_list_catalogs",
                Description: "List all available Databricks Unity Catalog catalogs",
                InputSchema: map[string]any{
                    "type":       "object",
                    "properties": map[string]any{},
                },
            },
            fn: p.handleListCatalogs,
        },
        // Add other tools...
    }

    for _, t := range tools {
        if err := p.registry.Register(t.tool, t.fn); err != nil {
            return err
        }
    }

    return nil
}

func (p *Provider) handleListCatalogs(ctx context.Context, params json.RawMessage) (*mcp.ToolResult, error) {
    result, err := p.client.ListCatalogs(ctx)
    if err != nil {
        return nil, err
    }

    text := formatCatalogsResult(result)
    return &mcp.ToolResult{
        Content: []mcp.Content{{Type: "text", Text: text}},
    }, nil
}

func (p *Provider) ListTools(ctx context.Context) ([]mcp.Tool, error) {
    return p.registry.ListTools(), nil
}

func (p *Provider) CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.ToolResult, error) {
    return p.registry.Call(ctx, name, params)
}
```

### 4.7 Add Result Formatting

**pkg/providers/databricks/format.go:**

```go
func formatCatalogsResult(result *ListCatalogsResult) string
func formatSchemasResult(result *ListSchemasResult) string
func formatTablesResult(result *ListTablesResult) string
func formatTableDetails(result *TableDetails) string
func formatQueryResult(rows []map[string]any) string
```

### 4.8 Write Tests

**pkg/providers/databricks/client_test.go:**

Use mock Databricks client or integration tests with real workspace:

```go
func TestClient_ListCatalogs(t *testing.T)
func TestClient_ListSchemas(t *testing.T)
func TestClient_ListTables(t *testing.T)
func TestClient_DescribeTable(t *testing.T)
func TestClient_ExecuteQuery(t *testing.T)

// Integration tests (require DATABRICKS_HOST, DATABRICKS_TOKEN)
func TestIntegration_RealWorkspace(t *testing.T) {
    if os.Getenv("DATABRICKS_HOST") == "" {
        t.Skip("Skipping integration test: DATABRICKS_HOST not set")
    }
    // ...
}
```

## Acceptance Criteria

- [ ] Databricks SDK integrated
- [ ] List catalogs/schemas/tables working
- [ ] Table description with metadata and sample data
- [ ] SQL query execution with result parsing
- [ ] Provider registered with MCP registry
- [ ] Result formatting produces readable output
- [ ] Unit tests pass
- [ ] Integration tests pass (with real workspace)

## Testing Commands

```bash
# Unit tests
go test ./pkg/providers/databricks/...

# Integration tests (requires env vars)
export DATABRICKS_HOST=your-workspace.cloud.databricks.com
export DATABRICKS_TOKEN=dapi...
export DATABRICKS_WAREHOUSE_ID=abc123
go test -tags=integration ./pkg/providers/databricks/...
```

## Next Steps

Proceed to Step 5: I/O Provider once all acceptance criteria are met.
