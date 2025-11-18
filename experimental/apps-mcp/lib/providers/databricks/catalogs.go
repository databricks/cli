package databricks

import (
	"context"
	"fmt"
	"strings"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// ListCatalogsResult represents the result of listing catalogs
type ListCatalogsResult struct {
	Catalogs []string `json:"catalogs"`
}

// ListCatalogs lists all available Databricks Unity Catalog catalogs
func ListCatalogs(ctx context.Context, cfg *mcp.Config) (*ListCatalogsResult, error) {
	w := cmdctx.WorkspaceClient(ctx)
	catalogs, err := w.Catalogs.ListAll(ctx, catalog.ListCatalogsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}

	names := make([]string, len(catalogs))
	for i, cat := range catalogs {
		names[i] = cat.Name
	}

	return &ListCatalogsResult{Catalogs: names}, nil
}

// ListSchemasArgs represents arguments for listing schemas
type ListSchemasArgs struct {
	CatalogName string `json:"catalog_name"`
	Filter      string `json:"filter,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// ListSchemasResult represents the result of listing schemas
type ListSchemasResult struct {
	Schemas    []string `json:"schemas"`
	TotalCount int      `json:"total_count"`
	ShownCount int      `json:"shown_count"`
	Offset     int      `json:"offset"`
	Limit      int      `json:"limit"`
}

// ListSchemas lists schemas in a catalog with optional filtering and pagination
func ListSchemas(ctx context.Context, cfg *mcp.Config, args *ListSchemasArgs) (*ListSchemasResult, error) {
	if args.Limit == 0 {
		args.Limit = 1000
	}

	w := cmdctx.WorkspaceClient(ctx)
	schemas, err := w.Schemas.ListAll(ctx, catalog.ListSchemasRequest{
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
		var filtered []string
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

// ListTablesArgs represents arguments for listing tables
type ListTablesArgs struct {
	CatalogName         string `json:"catalog_name"`
	SchemaName          string `json:"schema_name"`
	ExcludeInaccessible bool   `json:"exclude_inaccessible"`
}

// TableInfo represents information about a table
type TableInfo struct {
	Name        string  `json:"name"`
	CatalogName string  `json:"catalog_name"`
	SchemaName  string  `json:"schema_name"`
	FullName    string  `json:"full_name"`
	TableType   string  `json:"table_type"`
	Owner       *string `json:"owner,omitempty"`
	Comment     *string `json:"comment,omitempty"`
}

// ListTablesResult represents the result of listing tables
type ListTablesResult struct {
	Tables []TableInfo `json:"tables"`
}

// ListTables lists tables in a schema
func ListTables(ctx context.Context, cfg *mcp.Config, args *ListTablesArgs) (*ListTablesResult, error) {
	w := cmdctx.WorkspaceClient(ctx)
	tables, err := w.Tables.ListAll(ctx, catalog.ListTablesRequest{
		CatalogName:   args.CatalogName,
		SchemaName:    args.SchemaName,
		IncludeBrowse: !args.ExcludeInaccessible,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	infos := make([]TableInfo, len(tables))
	for i, table := range tables {
		var owner, comment *string
		if table.Owner != "" {
			owner = &table.Owner
		}
		if table.Comment != "" {
			comment = &table.Comment
		}
		infos[i] = TableInfo{
			Name:        table.Name,
			CatalogName: table.CatalogName,
			SchemaName:  table.SchemaName,
			FullName:    fmt.Sprintf("%s.%s.%s", table.CatalogName, table.SchemaName, table.Name),
			TableType:   string(table.TableType),
			Owner:       owner,
			Comment:     comment,
		}
	}

	return &ListTablesResult{Tables: infos}, nil
}
