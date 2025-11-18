package databricks

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
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
		MaxResults:  args.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	names := make([]string, len(schemas))
	for i, schema := range schemas {
		names[i] = schema.Name
	}

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
	CatalogName         string  `json:"catalog_name"`
	SchemaName          string  `json:"schema_name"`
	ExcludeInaccessible bool    `json:"exclude_inaccessible"`
	PageSize            *int    `json:"page_size,omitempty"`
	PageToken           *string `json:"page_token,omitempty"`
	Filter              *string `json:"filter,omitempty"`
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
	Tables        []TableInfo `json:"tables"`
	NextPageToken *string     `json:"next_page_token,omitempty"`
	TotalCount    int         `json:"total_count"`
}

// matchFilter checks if a name matches a filter pattern (supports wildcards)
func matchFilter(name, filter string) bool {
	name = strings.ToLower(name)
	filter = strings.ToLower(filter)

	if strings.Contains(filter, "*") {
		parts := strings.Split(filter, "*")
		pos := 0
		for i, part := range parts {
			if part == "" {
				continue
			}
			idx := strings.Index(name[pos:], part)
			if idx == -1 {
				return false
			}
			if i == 0 && idx != 0 {
				return false
			}
			pos += idx + len(part)
		}
		if !strings.HasSuffix(filter, "*") && !strings.HasSuffix(name, parts[len(parts)-1]) {
			return false
		}
		return true
	}

	return strings.Contains(name, filter)
}

// ListTables lists tables in a schema with pagination support
func ListTables(ctx context.Context, cfg *mcp.Config, args *ListTablesArgs) (*ListTablesResult, error) {
	pageSize := 100
	if args.PageSize != nil {
		pageSize = min(*args.PageSize, 1000)
	}

	w := cmdctx.WorkspaceClient(ctx)

	apiPath := "/api/2.1/unity-catalog/tables"
	params := url.Values{}
	params.Add("catalog_name", args.CatalogName)
	params.Add("schema_name", args.SchemaName)
	if !args.ExcludeInaccessible {
		params.Add("include_browse", "true")
	}
	params.Add("max_results", strconv.Itoa(pageSize))
	if args.PageToken != nil && *args.PageToken != "" {
		params.Add("page_token", *args.PageToken)
	}

	fullPath := apiPath + "?" + params.Encode()
	clientCfg, err := config.HTTPClientConfigFromConfig(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client config: %w", err)
	}
	apiClient := httpclient.NewApiClient(clientCfg)

	var response catalog.ListTablesResponse
	err = apiClient.Do(ctx, "GET", fullPath, httpclient.WithResponseUnmarshal(&response))
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	var infos []TableInfo
	for _, table := range response.Tables {
		if args.Filter != nil && !matchFilter(table.Name, *args.Filter) {
			continue
		}

		var owner, comment *string
		if table.Owner != "" {
			owner = &table.Owner
		}
		if table.Comment != "" {
			comment = &table.Comment
		}

		infos = append(infos, TableInfo{
			Name:        table.Name,
			CatalogName: table.CatalogName,
			SchemaName:  table.SchemaName,
			FullName:    fmt.Sprintf("%s.%s.%s", table.CatalogName, table.SchemaName, table.Name),
			TableType:   string(table.TableType),
			Owner:       owner,
			Comment:     comment,
		})
	}

	var nextToken *string
	if response.NextPageToken != "" {
		nextToken = &response.NextPageToken
	}

	return &ListTablesResult{
		Tables:        infos,
		NextPageToken: nextToken,
		TotalCount:    len(infos),
	}, nil
}
