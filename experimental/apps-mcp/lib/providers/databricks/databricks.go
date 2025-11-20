package databricks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

const (
	DefaultWaitTimeout = 30 * time.Second
	MaxPollAttempts    = 30
	DefaultLimit       = 500
	DefaultSampleSize  = 5
	DefaultMaxRows     = 1000
	MaxMaxRows         = 10000
	PollInterval       = 2 * time.Second
	MaxRowDisplayLimit = 100
	SessionClientKey   = "databricks_client"
)

// ============================================================================
// Authentication Functions
// ============================================================================

// ConfigureAuth creates and validates a Databricks workspace client with optional host and profile.
// The authenticated client is stored in the session data for reuse across tool calls.
func ConfigureAuth(ctx context.Context, sess *session.Session, host, profile *string) (*databricks.WorkspaceClient, error) {
	// Skip auth check if testing
	if os.Getenv("DATABRICKS_MCP_SKIP_AUTH_CHECK") == "1" {
		return nil, nil
	}

	var cfg *databricks.Config
	if host != nil || profile != nil {
		cfg = &databricks.Config{}
		if host != nil {
			cfg.Host = *host
		}
		if profile != nil {
			cfg.Profile = *profile
		}
	}

	var client *databricks.WorkspaceClient
	var err error
	if cfg != nil {
		client, err = databricks.NewWorkspaceClient(cfg)
	} else {
		client, err = databricks.NewWorkspaceClient()
	}
	if err != nil {
		return nil, err
	}

	_, err = client.CurrentUser.Me(ctx)
	if err != nil {
		if profile == nil {
			return nil, errors.New(prompts.MustExecuteTemplate("auth_u2m.tmpl", map[string]string{
				"WorkspaceURL": *host,
			}))
		}
		return nil, wrapAuthError(err)
	}

	// Store client in session data
	sess.Set(middlewares.DatabricksClientKey, client)

	return client, nil
}

// wrapAuthError wraps configuration errors with helpful messages
func wrapAuthError(err error) error {
	if errors.Is(err, config.ErrCannotConfigureDefault) {
		return errors.New(prompts.MustExecuteTemplate("auth_error.tmpl", nil))
	}
	return err
}

// ============================================================================
// Helper Functions
// ============================================================================

// applyPagination applies limit and offset to a slice and returns paginated results with counts
func applyPagination[T any](items []T, limit, offset int) ([]T, int, int) {
	total := len(items)
	start := min(offset, total)
	end := min(start+limit, total)
	paginated := items[start:end]
	shown := len(paginated)
	return paginated, total, shown
}

// validateIdentifier validates that an identifier (catalog, schema, table name) contains only safe characters.
// Allows alphanumeric, underscore, hyphen, and dot (for qualified names).
func validateIdentifier(id string) error {
	if id == "" {
		return errors.New("identifier cannot be empty")
	}

	// Allow alphanumeric, underscore, hyphen, and dot for qualified names
	for _, ch := range id {
		isValid := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch == '.'

		if !isValid {
			return fmt.Errorf("invalid identifier '%s': contains unsafe characters", id)
		}
	}

	return nil
}

// escapeLikePattern escapes a user-provided pattern for safe use in SQL LIKE clauses.
// - Escapes the backslash escape character itself
// - Escapes SQL wildcards (%, _) to treat them as literals
// - Converts glob-style wildcards (* and ?) to SQL wildcards
// Must be used with ESCAPE '\\' clause in SQL query.
func escapeLikePattern(input string) string {
	result := strings.ReplaceAll(input, `\`, `\\`) // escape the escape char first!
	result = strings.ReplaceAll(result, `%`, `\%`) // escape SQL wildcard %
	result = strings.ReplaceAll(result, `_`, `\_`) // escape SQL wildcard _
	result = strings.ReplaceAll(result, `*`, `%`)  // convert glob * to SQL %
	result = strings.ReplaceAll(result, `?`, `_`)  // convert glob ? to SQL _
	return result
}

// ============================================================================
// Argument Types (shared between agent and MCP)
// ============================================================================

type DatabricksListCatalogsArgs struct {
	// no parameters needed - lists all available catalogs
}

type DatabricksListSchemasArgs struct {
	CatalogName string  `json:"catalog_name"`
	Filter      *string `json:"filter,omitempty"`
	Limit       int     `json:"limit,omitempty"`
	Offset      int     `json:"offset,omitempty"`
}

type DatabricksListTablesArgs struct {
	// Optional catalog name. If omitted, searches across all catalogs.
	CatalogName *string `json:"catalog_name,omitempty"`
	// Optional schema name. If omitted, searches across all schemas (requires catalog_name to also be omitted).
	SchemaName *string `json:"schema_name,omitempty"`
	// Optional filter pattern for table name (supports wildcards when searching across catalogs/schemas)
	Filter *string `json:"filter,omitempty"`
	Limit  int     `json:"limit,omitempty"`
	Offset int     `json:"offset,omitempty"`
}

type DatabricksDescribeTableArgs struct {
	TableFullName string `json:"table_full_name"`
	SampleSize    int    `json:"sample_size,omitempty"`
}

type DatabricksExecuteQueryArgs struct {
	Query string `json:"query"`
}

// ============================================================================
// Request Types (internal to client)
// ============================================================================

type ExecuteSqlRequest struct {
	Query string `json:"query"`
}

type ListSchemasRequest struct {
	CatalogName string  `json:"catalog_name"`
	Filter      *string `json:"filter,omitempty"`
	Limit       int     `json:"limit,omitempty"`
	Offset      int     `json:"offset,omitempty"`
}

type ListTablesRequest struct {
	CatalogName *string `json:"catalog_name,omitempty"`
	SchemaName  *string `json:"schema_name,omitempty"`
	Filter      *string `json:"filter,omitempty"`
	Limit       int     `json:"limit,omitempty"`
	Offset      int     `json:"offset,omitempty"`
}

type DescribeTableRequest struct {
	TableFullName string `json:"table_full_name"`
	SampleSize    int    `json:"sample_size,omitempty"`
}

// ============================================================================
// Response Types
// ============================================================================

type TableDetailsResponse struct {
	FullName         string                   `json:"full_name"`
	TableType        string                   `json:"table_type"`
	Owner            *string                  `json:"owner,omitempty"`
	Comment          *string                  `json:"comment,omitempty"`
	StorageLocation  *string                  `json:"storage_location,omitempty"`
	DataSourceFormat *string                  `json:"data_source_format,omitempty"`
	Columns          []ColumnMetadataResponse `json:"columns"`
	SampleData       []map[string]any         `json:"sample_data,omitempty"`
	RowCount         *int64                   `json:"row_count,omitempty"`
}

type ColumnMetadataResponse struct {
	Name     string  `json:"name"`
	DataType string  `json:"data_type"`
	Comment  *string `json:"comment,omitempty"`
	Nullable bool    `json:"nullable"`
}

type TableInfoResponse struct {
	Name        string  `json:"name"`
	CatalogName string  `json:"catalog_name"`
	SchemaName  string  `json:"schema_name"`
	FullName    string  `json:"full_name"`
	TableType   string  `json:"table_type"`
	Owner       *string `json:"owner,omitempty"`
	Comment     *string `json:"comment,omitempty"`
}

type ListCatalogsResultResponse struct {
	Catalogs []string `json:"catalogs"`
}

type ListSchemasResultResponse struct {
	Schemas    []string `json:"schemas"`
	TotalCount int      `json:"total_count"`
	ShownCount int      `json:"shown_count"`
	Offset     int      `json:"offset"`
	Limit      int      `json:"limit"`
}

type ListTablesResultResponse struct {
	Tables     []TableInfoResponse `json:"tables"`
	TotalCount int                 `json:"total_count"`
	ShownCount int                 `json:"shown_count"`
	Offset     int                 `json:"offset"`
	Limit      int                 `json:"limit"`
}

type ExecuteSqlResultResponse struct {
	Rows []map[string]any `json:"rows"`
}

// ============================================================================
// Display Trait for Tool Results
// ============================================================================

func (r *ListCatalogsResultResponse) Display() string {
	if len(r.Catalogs) == 0 {
		return "No catalogs found."
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d catalogs:", len(r.Catalogs)))
	lines = append(lines, "")

	for _, catalog := range r.Catalogs {
		lines = append(lines, "• "+catalog)
	}

	return strings.Join(lines, "\n")
}

func (r *ListSchemasResultResponse) Display() string {
	if len(r.Schemas) == 0 {
		return "No schemas found."
	}

	var lines []string
	lines = append(lines,
		fmt.Sprintf("Showing %d of %d schemas (offset: %d, limit: %d):",
			r.ShownCount, r.TotalCount, r.Offset, r.Limit))
	lines = append(lines, "")

	for _, schema := range r.Schemas {
		lines = append(lines, "• "+schema)
	}

	return strings.Join(lines, "\n")
}

func (r *ListTablesResultResponse) Display() string {
	if len(r.Tables) == 0 {
		return "No tables found."
	}

	var lines []string
	lines = append(lines,
		fmt.Sprintf("Showing %d of %d tables (offset: %d, limit: %d):",
			r.ShownCount, r.TotalCount, r.Offset, r.Limit))
	lines = append(lines, "")

	for _, table := range r.Tables {
		info := fmt.Sprintf("• %s (%s)", table.FullName, table.TableType)
		if table.Owner != nil {
			info += " - Owner: " + *table.Owner
		}
		if table.Comment != nil {
			info += " - " + *table.Comment
		}
		lines = append(lines, info)
	}

	return strings.Join(lines, "\n")
}

func (r *TableDetailsResponse) Display() string {
	var lines []string

	lines = append(lines, "Table: "+r.FullName)
	lines = append(lines, "Table Type: "+r.TableType)

	if r.Owner != nil {
		lines = append(lines, "Owner: "+*r.Owner)
	}
	if r.Comment != nil {
		lines = append(lines, "Comment: "+*r.Comment)
	}
	if r.RowCount != nil {
		lines = append(lines, fmt.Sprintf("Row Count: %d", *r.RowCount))
	}
	if r.StorageLocation != nil {
		lines = append(lines, "Storage Location: "+*r.StorageLocation)
	}
	if r.DataSourceFormat != nil {
		lines = append(lines, "Data Source Format: "+*r.DataSourceFormat)
	}

	if len(r.Columns) > 0 {
		lines = append(lines, fmt.Sprintf("\nColumns (%d):", len(r.Columns)))
		for _, col := range r.Columns {
			nullableStr := "nullable"
			if !col.Nullable {
				nullableStr = "required"
			}
			colInfo := fmt.Sprintf("  - %s: %s (%s)", col.Name, col.DataType, nullableStr)
			if col.Comment != nil {
				colInfo += " - " + *col.Comment
			}
			lines = append(lines, colInfo)
		}
	}

	if len(r.SampleData) > 0 {
		lines = append(lines, fmt.Sprintf("\nSample Data (%d rows):", len(r.SampleData)))
		displayCount := min(len(r.SampleData), 5)
		for i := range displayCount {
			row := r.SampleData[i]
			var rowParts []string
			for k, v := range row {
				rowParts = append(rowParts, fmt.Sprintf("%s: %v", k, formatValue(v)))
			}
			lines = append(lines, fmt.Sprintf("  Row %d: %s", i+1, strings.Join(rowParts, ", ")))
		}
		if len(r.SampleData) > 5 {
			lines = append(lines, "...")
		}
	}

	return strings.Join(lines, "\n")
}

func (r *ExecuteSqlResultResponse) Display() string {
	if len(r.Rows) == 0 {
		return "Query executed successfully but returned no results."
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Query returned %d rows:", len(r.Rows)))
	lines = append(lines, "")

	if len(r.Rows) > 0 {
		var columns []string
		for k := range r.Rows[0] {
			columns = append(columns, k)
		}
		lines = append(lines, "Columns: "+strings.Join(columns, ", "))
		lines = append(lines, "")
		lines = append(lines, "Results:")
	}

	limit := min(len(r.Rows), MaxRowDisplayLimit)
	for i := range limit {
		row := r.Rows[i]
		var rowParts []string
		for k, v := range row {
			rowParts = append(rowParts, fmt.Sprintf("%s: %v", k, formatValue(v)))
		}
		lines = append(lines, fmt.Sprintf("  Row %d: %s", i+1, strings.Join(rowParts, ", ")))
	}

	if len(r.Rows) > MaxRowDisplayLimit {
		lines = append(lines, fmt.Sprintf("\n... showing first %d of %d total rows",
			MaxRowDisplayLimit, len(r.Rows)))
	}

	return strings.Join(lines, "\n")
}

func formatValue(value any) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%v", value)
}

// ============================================================================
// DatabricksRestClient
// ============================================================================

type DatabricksRestClient struct {
	client      *databricks.WorkspaceClient
	warehouseID string
}

// NewDatabricksRestClient creates a new Databricks REST client using the SDK
func NewDatabricksRestClient(ctx context.Context, cfg *mcp.Config) (*DatabricksRestClient, error) {
	client := middlewares.MustGetDatabricksClient(ctx)

	warehouseID, err := middlewares.GetWarehouseID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get warehouse ID: %w", err)
	}

	return &DatabricksRestClient{
		client:      client,
		warehouseID: warehouseID,
	}, nil
}

// ExecuteSql executes a SQL query and returns the results
func (c *DatabricksRestClient) ExecuteSql(ctx context.Context, request *ExecuteSqlRequest) (*ExecuteSqlResultResponse, error) {
	rows, err := c.executeSqlImpl(ctx, request.Query)
	if err != nil {
		return nil, err
	}
	return &ExecuteSqlResultResponse{Rows: rows}, nil
}

// executeSqlWithParams executes SQL with named parameters for safe dynamic queries
func (c *DatabricksRestClient) executeSqlWithParams(ctx context.Context, query string, parameters []sql.StatementParameterListItem) ([]map[string]any, error) {
	result, err := c.client.StatementExecution.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
		Statement:   query,
		Parameters:  parameters,
		WarehouseId: c.warehouseID,
		WaitTimeout: fmt.Sprintf("%ds", int(DefaultWaitTimeout.Seconds())),
		Format:      sql.FormatJsonArray,
		Disposition: sql.DispositionInline,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement: %w", err)
	}

	// Check if we need to poll for results
	if result.Status != nil {
		state := result.Status.State
		switch state {
		case sql.StatementStatePending, sql.StatementStateRunning:
			return c.pollForResults(ctx, result.StatementId)
		case sql.StatementStateFailed:
			errMsg := "unknown error"
			if result.Status.Error != nil && result.Status.Error.Message != "" {
				errMsg = result.Status.Error.Message
			}
			return nil, fmt.Errorf("SQL execution failed: %s", errMsg)
		case sql.StatementStateCanceled, sql.StatementStateClosed, sql.StatementStateSucceeded:
			break
		}
	}

	return c.processStatementResult(ctx, result)
}

func (c *DatabricksRestClient) executeSqlImpl(ctx context.Context, query string) ([]map[string]any, error) {
	result, err := c.client.StatementExecution.ExecuteStatement(ctx, sql.ExecuteStatementRequest{
		Statement:     query,
		WarehouseId:   c.warehouseID,
		WaitTimeout:   fmt.Sprintf("%ds", int(DefaultWaitTimeout.Seconds())),
		OnWaitTimeout: sql.ExecuteStatementRequestOnWaitTimeoutContinue,
		Format:        sql.FormatJsonArray,
		Disposition:   sql.DispositionInline,
		RowLimit:      100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement: %w", err)
	}

	// Check if we need to poll for results
	if result.Status != nil {
		state := result.Status.State
		switch state {
		case sql.StatementStatePending, sql.StatementStateRunning:
			return c.pollForResults(ctx, result.StatementId)
		case sql.StatementStateFailed:
			errMsg := "unknown error"
			if result.Status.Error != nil && result.Status.Error.Message != "" {
				errMsg = result.Status.Error.Message
			}
			return nil, fmt.Errorf("SQL execution failed: %s", errMsg)
		case sql.StatementStateCanceled, sql.StatementStateClosed, sql.StatementStateSucceeded:
			break
		}
	}

	return c.processStatementResult(ctx, result)
}

func (c *DatabricksRestClient) pollForResults(ctx context.Context, statementID string) ([]map[string]any, error) {
	for attempt := range MaxPollAttempts {
		log.Debugf(ctx, "Polling attempt %d for statement %s", attempt+1, statementID)

		result, err := c.client.StatementExecution.GetStatement(ctx, sql.GetStatementRequest{
			StatementId: statementID,
		})
		if err != nil {
			return nil, fmt.Errorf("polling attempt %d failed: %w", attempt+1, err)
		}

		if result.Status != nil {
			switch result.Status.State {
			case sql.StatementStateSucceeded:
				return c.processStatementResult(ctx, result)
			case sql.StatementStateFailed:
				errMsg := "unknown error"
				if result.Status.Error != nil && result.Status.Error.Message != "" {
					errMsg = result.Status.Error.Message
				}
				return nil, fmt.Errorf("SQL execution failed: %s", errMsg)
			case sql.StatementStatePending, sql.StatementStateRunning:
				time.Sleep(PollInterval)
				continue
			default:
				return nil, fmt.Errorf("unexpected statement state: %s", result.Status.State)
			}
		}
	}

	return nil, fmt.Errorf("polling timeout exceeded for statement %s", statementID)
}

func (c *DatabricksRestClient) processStatementResult(ctx context.Context, result *sql.StatementResponse) ([]map[string]any, error) {
	log.Debugf(ctx, "Processing statement result: %+v", result)

	if result.Manifest == nil || result.Manifest.Schema == nil {
		log.Debugf(ctx, "No schema in response")
		return nil, errors.New("no schema in response")
	}

	schema := result.Manifest.Schema

	// Check if statement returns no result set (DDL, DML writes, etc.)
	if len(schema.Columns) == 0 {
		log.Debugf(ctx, "Statement executed successfully (no result set)")
		return []map[string]any{}, nil
	}

	// Try to get inline data
	if result.Result != nil && result.Result.DataArray != nil {
		log.Debugf(ctx, "Found %d rows of inline data", len(result.Result.DataArray))
		return c.processDataArray(schema, result.Result.DataArray)
	}

	// Query executed successfully but returned 0 rows (empty result set is valid)
	log.Debugf(ctx, "Query executed successfully with empty result set")
	return []map[string]any{}, nil
}

func (c *DatabricksRestClient) processDataArray(schema *sql.ResultSchema, dataArray [][]string) ([]map[string]any, error) {
	var results []map[string]any

	for _, row := range dataArray {
		rowMap := make(map[string]any)

		for i, column := range schema.Columns {
			var value any
			if i < len(row) {
				value = row[i]
			}
			rowMap[column.Name] = value
		}

		results = append(results, rowMap)
	}

	return results, nil
}

// ListCatalogs lists all available Databricks Unity Catalog catalogs
func (c *DatabricksRestClient) ListCatalogs(ctx context.Context) (*ListCatalogsResultResponse, error) {
	catalogs, err := c.listCatalogsImpl(ctx)
	if err != nil {
		return nil, err
	}
	return &ListCatalogsResultResponse{Catalogs: catalogs}, nil
}

func (c *DatabricksRestClient) listCatalogsImpl(ctx context.Context) ([]string, error) {
	var allCatalogs []string

	iter := c.client.Catalogs.List(ctx, catalog.ListCatalogsRequest{})
	for iter.HasNext(ctx) {
		cat, err := iter.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to iterate catalogs: %w", err)
		}
		allCatalogs = append(allCatalogs, cat.Name)
	}

	return allCatalogs, nil
}

// ListSchemas lists schemas in a catalog with optional filtering and pagination
func (c *DatabricksRestClient) ListSchemas(ctx context.Context, request *ListSchemasRequest) (*ListSchemasResultResponse, error) {
	schemas, err := c.listSchemasImpl(ctx, request.CatalogName)
	if err != nil {
		return nil, err
	}

	// Apply filter if provided
	if request.Filter != nil {
		filterLower := strings.ToLower(*request.Filter)
		var filtered []string
		for _, s := range schemas {
			if strings.Contains(strings.ToLower(s), filterLower) {
				filtered = append(filtered, s)
			}
		}
		schemas = filtered
	}

	limit := request.Limit
	if limit == 0 {
		limit = DefaultLimit
	}

	paginated, totalCount, shownCount := applyPagination(schemas, limit, request.Offset)

	return &ListSchemasResultResponse{
		Schemas:    paginated,
		TotalCount: totalCount,
		ShownCount: shownCount,
		Offset:     request.Offset,
		Limit:      limit,
	}, nil
}

func (c *DatabricksRestClient) listSchemasImpl(ctx context.Context, catalogName string) ([]string, error) {
	var allSchemas []string

	iter := c.client.Schemas.List(ctx, catalog.ListSchemasRequest{
		CatalogName: catalogName,
	})
	for iter.HasNext(ctx) {
		schema, err := iter.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to iterate schemas: %w", err)
		}
		allSchemas = append(allSchemas, schema.Name)
	}

	return allSchemas, nil
}

// ListTables lists tables with support for wildcard searches and pagination
func (c *DatabricksRestClient) ListTables(ctx context.Context, request *ListTablesRequest) (*ListTablesResultResponse, error) {
	if request.CatalogName != nil && request.SchemaName != nil {
		// Fast path - use REST API for specific catalog/schema
		tables, err := c.listTablesImpl(ctx, *request.CatalogName, *request.SchemaName, true)
		if err != nil {
			return nil, err
		}

		// Apply filter if provided
		if request.Filter != nil {
			filterLower := strings.ToLower(*request.Filter)
			var filtered []TableInfoResponse
			for _, t := range tables {
				// Match against both table name and schema name
				if strings.Contains(strings.ToLower(t.Name), filterLower) ||
					strings.Contains(strings.ToLower(t.SchemaName), filterLower) {
					filtered = append(filtered, t)
				}
			}
			tables = filtered
		}

		limit := request.Limit
		if limit == 0 {
			limit = DefaultLimit
		}

		paginated, totalCount, shownCount := applyPagination(tables, limit, request.Offset)

		return &ListTablesResultResponse{
			Tables:     paginated,
			TotalCount: totalCount,
			ShownCount: shownCount,
			Offset:     request.Offset,
			Limit:      limit,
		}, nil
	}

	// Wildcard search - use system.information_schema.tables
	return c.listTablesViaInformationSchema(ctx, request)
}

// listTablesViaInformationSchema searches tables across catalogs/schemas using system.information_schema
func (c *DatabricksRestClient) listTablesViaInformationSchema(ctx context.Context, request *ListTablesRequest) (*ListTablesResultResponse, error) {
	// Validate invalid combination
	if request.CatalogName == nil && request.SchemaName != nil {
		return nil, errors.New("schema_name requires catalog_name to be specified")
	}

	// Validate identifiers for SQL safety
	if request.CatalogName != nil {
		if err := validateIdentifier(*request.CatalogName); err != nil {
			return nil, err
		}
	}
	if request.SchemaName != nil {
		if err := validateIdentifier(*request.SchemaName); err != nil {
			return nil, err
		}
	}

	// Build WHERE conditions with parameterized queries
	var conditions []string
	var parameters []sql.StatementParameterListItem

	if request.CatalogName != nil {
		conditions = append(conditions, "table_catalog = :catalog")
		parameters = append(parameters, sql.StatementParameterListItem{
			Name:  "catalog",
			Value: *request.CatalogName,
			Type:  "STRING",
		})
	}

	if request.SchemaName != nil {
		conditions = append(conditions, "table_schema = :schema")
		parameters = append(parameters, sql.StatementParameterListItem{
			Name:  "schema",
			Value: *request.SchemaName,
			Type:  "STRING",
		})
	}

	if request.Filter != nil {
		// Use dedicated escape function for LIKE patterns
		pattern := escapeLikePattern(*request.Filter)

		// Wrap pattern for substring match if no wildcards at boundaries
		if !strings.HasPrefix(pattern, "%") && !strings.HasSuffix(pattern, "%") {
			pattern = "%" + pattern + "%"
		}

		// Match against both table name and schema name
		conditions = append(conditions, "(table_name LIKE :pattern ESCAPE '\\\\' OR table_schema LIKE :pattern ESCAPE '\\\\')")
		parameters = append(parameters, sql.StatementParameterListItem{
			Name:  "pattern",
			Value: pattern,
			Type:  "STRING",
		})
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := request.Limit
	if limit == 0 {
		limit = DefaultLimit
	}

	// Build SQL query with parameter markers
	query := fmt.Sprintf(`
		SELECT table_catalog, table_schema, table_name, table_type
		FROM system.information_schema.tables
		%s
		ORDER BY table_catalog, table_schema, table_name
		LIMIT %d
	`, whereClause, limit+request.Offset)

	// Execute query with parameters
	rows, err := c.executeSqlWithParams(ctx, query, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to query information_schema: %w", err)
	}

	// Parse results into TableInfo with explicit error handling
	var tables []TableInfoResponse
	for _, row := range rows {
		catalogVal, ok := row["table_catalog"]
		if !ok {
			return nil, errors.New("missing table_catalog in row")
		}
		catalog := fmt.Sprintf("%v", catalogVal)

		schemaVal, ok := row["table_schema"]
		if !ok {
			return nil, errors.New("missing table_schema in row")
		}
		schema := fmt.Sprintf("%v", schemaVal)

		nameVal, ok := row["table_name"]
		if !ok {
			return nil, errors.New("missing table_name in row")
		}
		name := fmt.Sprintf("%v", nameVal)

		tableTypeVal, ok := row["table_type"]
		if !ok {
			return nil, errors.New("missing table_type in row")
		}
		tableType := fmt.Sprintf("%v", tableTypeVal)

		tables = append(tables, TableInfoResponse{
			Name:        name,
			CatalogName: catalog,
			SchemaName:  schema,
			TableType:   tableType,
			FullName:    fmt.Sprintf("%s.%s.%s", catalog, schema, name),
			Owner:       nil,
			Comment:     nil,
		})
	}

	paginated, totalCount, shownCount := applyPagination(tables, limit, request.Offset)

	return &ListTablesResultResponse{
		Tables:     paginated,
		TotalCount: totalCount,
		ShownCount: shownCount,
		Offset:     request.Offset,
		Limit:      limit,
	}, nil
}

func (c *DatabricksRestClient) listTablesImpl(ctx context.Context, catalogName, schemaName string, excludeInaccessible bool) ([]TableInfoResponse, error) {
	var tables []TableInfoResponse

	apiClient, err := middlewares.MustGetApiClient(ctx)
	if err != nil {
		return nil, err
	}

	nextPageToken := ""
	for {
		apiPath := "/api/2.1/unity-catalog/tables"
		params := url.Values{}
		params.Add("catalog_name", catalogName)
		params.Add("schema_name", schemaName)
		if excludeInaccessible {
			params.Add("include_browse", "false")
		}
		params.Add("max_results", strconv.Itoa(DefaultLimit))
		if nextPageToken != "" {
			params.Add("page_token", nextPageToken)
		}
		fullPath := apiPath + "?" + params.Encode()

		var response catalog.ListTablesResponse
		err = apiClient.Do(ctx, "GET", fullPath, httpclient.WithResponseUnmarshal(&response))
		if err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}

		for _, table := range response.Tables {
			tables = append(tables, TableInfoResponse{
				Name:        table.Name,
				CatalogName: table.CatalogName,
				SchemaName:  table.SchemaName,
				TableType:   string(table.TableType),
				FullName:    fmt.Sprintf("%s.%s.%s", table.CatalogName, table.SchemaName, table.Name),
				Owner:       &table.Owner,
				Comment:     &table.Comment,
			})
		}

		if response.NextPageToken != "" {
			nextPageToken = response.NextPageToken
		} else {
			break
		}
	}

	return tables, nil
}

// DescribeTable retrieves detailed information about a table including metadata and sample data
func (c *DatabricksRestClient) DescribeTable(ctx context.Context, request *DescribeTableRequest) (*TableDetailsResponse, error) {
	sampleRows := request.SampleSize
	if sampleRows == 0 {
		sampleRows = DefaultSampleSize
	}
	tableName := request.TableFullName

	// Get basic table metadata from Unity Catalog
	tableResponse, err := c.client.Tables.Get(ctx, catalog.GetTableRequest{
		FullName: tableName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get table metadata: %w", err)
	}

	// Build column metadata
	columns := make([]ColumnMetadataResponse, len(tableResponse.Columns))
	for i, col := range tableResponse.Columns {
		var comment *string
		if col.Comment != "" {
			comment = &col.Comment
		}

		columns[i] = ColumnMetadataResponse{
			Name:     col.Name,
			DataType: string(col.TypeName),
			Comment:  comment,
			Nullable: col.Nullable,
		}
	}

	// Get sample data and row count
	var sampleData []map[string]any
	if sampleRows > 0 {
		query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, sampleRows)
		if rows, err := c.executeSqlImpl(ctx, query); err == nil {
			sampleData = rows
		}
	}

	var rowCount *int64
	countQuery := "SELECT COUNT(*) as count FROM " + tableName
	if rows, err := c.executeSqlImpl(ctx, countQuery); err == nil && len(rows) > 0 {
		if countVal, ok := rows[0]["count"]; ok {
			switch v := countVal.(type) {
			case int64:
				rowCount = &v
			case float64:
				count := int64(v)
				rowCount = &count
			case string:
				// Try to parse string as int64
				var count int64
				if _, parseErr := fmt.Sscanf(v, "%d", &count); parseErr == nil {
					rowCount = &count
				}
			}
		}
	}

	var owner, comment, storageLocation, dataSourceFormat *string
	if tableResponse.Owner != "" {
		owner = &tableResponse.Owner
	}
	if tableResponse.Comment != "" {
		comment = &tableResponse.Comment
	}
	if tableResponse.StorageLocation != "" {
		storageLocation = &tableResponse.StorageLocation
	}
	if tableResponse.DataSourceFormat != "" {
		dsf := string(tableResponse.DataSourceFormat)
		dataSourceFormat = &dsf
	}

	tableType := "UNKNOWN"
	if tableResponse.TableType != "" {
		tableType = string(tableResponse.TableType)
	}

	return &TableDetailsResponse{
		FullName:         tableName,
		TableType:        tableType,
		Owner:            owner,
		Comment:          comment,
		StorageLocation:  storageLocation,
		DataSourceFormat: dataSourceFormat,
		Columns:          columns,
		SampleData:       sampleData,
		RowCount:         rowCount,
	}, nil
}
