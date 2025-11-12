package databricks

import (
	"context"
	"log/slog"

	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/providers"
	"github.com/databricks/cli/libs/mcp/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	providers.Register("databricks", func(cfg *mcp.Config, sess *session.Session, logger *slog.Logger) (providers.Provider, error) {
		return NewProvider(cfg, sess, logger)
	}, providers.ProviderConfig{
		Always: true,
	})
}

// Provider represents the Databricks provider that registers MCP tools
type Provider struct {
	client  *Client
	session *session.Session
	logger  *slog.Logger
}

// NewProvider creates a new Databricks provider
func NewProvider(cfg *mcp.Config, sess *session.Session, logger *slog.Logger) (*Provider, error) {
	client, err := NewClient(cfg, logger)
	if err != nil {
		return nil, err
	}

	return &Provider{
		client:  client,
		session: sess,
		logger:  logger,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "databricks"
}

// RegisterTools registers all Databricks tools with the MCP server
func (p *Provider) RegisterTools(server *mcp.Server) error {
	p.logger.Info("Registering Databricks tools")

	// Register databricks_list_catalogs
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "databricks_list_catalogs",
			Description: "List all available Databricks catalogs",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("databricks_list_catalogs called")

			result, err := p.client.ListCatalogs(ctx)
			if err != nil {
				return nil, nil, err
			}

			text := formatCatalogsResult(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register databricks_list_schemas
	type ListSchemasInput struct {
		CatalogName string `json:"catalog_name" jsonschema:"required" jsonschema_description:"Name of the catalog"`
		Filter      string `json:"filter,omitempty" jsonschema_description:"Optional filter string to search schema names"`
		Limit       int    `json:"limit,omitempty" jsonschema_description:"Maximum number of schemas to return (default: 1000)"`
		Offset      int    `json:"offset,omitempty" jsonschema_description:"Offset for pagination (default: 0)"`
	}

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "databricks_list_schemas",
			Description: "List all schemas in a Databricks catalog with pagination support",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args ListSchemasInput) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("databricks_list_schemas called", "catalog", args.CatalogName)

			listArgs := &ListSchemasArgs{
				CatalogName: args.CatalogName,
				Filter:      args.Filter,
				Limit:       args.Limit,
				Offset:      args.Offset,
			}

			result, err := p.client.ListSchemas(ctx, listArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatSchemasResult(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register databricks_list_tables
	type ListTablesInput struct {
		CatalogName         string `json:"catalog_name" jsonschema:"required" jsonschema_description:"Name of the catalog"`
		SchemaName          string `json:"schema_name" jsonschema:"required" jsonschema_description:"Name of the schema"`
		ExcludeInaccessible bool   `json:"exclude_inaccessible,omitempty" jsonschema_description:"Exclude inaccessible tables (default: false)"`
	}

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "databricks_list_tables",
			Description: "List tables in a Databricks catalog and schema",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args ListTablesInput) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("databricks_list_tables called", "catalog", args.CatalogName, "schema", args.SchemaName)

			listArgs := &ListTablesArgs{
				CatalogName:         args.CatalogName,
				SchemaName:          args.SchemaName,
				ExcludeInaccessible: args.ExcludeInaccessible,
			}

			result, err := p.client.ListTables(ctx, listArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatTablesResult(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register databricks_describe_table
	type DescribeTableInput struct {
		TableFullName string `json:"table_full_name" jsonschema:"required" jsonschema_description:"Full name of the table (catalog.schema.table)"`
		SampleSize    int    `json:"sample_size,omitempty" jsonschema_description:"Number of sample rows to return (default: 5)"`
	}

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "databricks_describe_table",
			Description: "Get detailed information about a Databricks table including schema and optional sample data",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args DescribeTableInput) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("databricks_describe_table called", "table", args.TableFullName)

			descArgs := &DescribeTableArgs{
				TableFullName: args.TableFullName,
				SampleSize:    args.SampleSize,
			}

			result, err := p.client.DescribeTable(ctx, descArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatTableDetails(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register databricks_execute_query
	type ExecuteQueryInput struct {
		Query string `json:"query" jsonschema:"required" jsonschema_description:"SQL query to execute"`
	}

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "databricks_execute_query",
			Description: "Execute SQL query in Databricks. Only single SQL statements are supported - do not send multiple statements separated by semicolons. For multiple statements, call this tool separately for each one. DO NOT create catalogs, schemas or tables - requires metastore admin privileges. Query existing data instead. Timeout: 60 seconds for query execution.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args ExecuteQueryInput) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("databricks_execute_query called", "query", args.Query)

			result, err := p.client.ExecuteQuery(ctx, args.Query)
			if err != nil {
				return nil, nil, err
			}

			text := formatQueryResult(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	p.logger.Info("Registered Databricks tools")
	return nil
}
