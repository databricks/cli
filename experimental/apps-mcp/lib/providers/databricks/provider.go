package databricks

import (
	"context"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	mcpsdk "github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/log"
)

func init() {
	providers.Register("databricks", func(ctx context.Context, cfg *mcp.Config, sess *session.Session) (providers.Provider, error) {
		return NewProvider(ctx, cfg, sess)
	}, providers.ProviderConfig{
		Always: true,
	})
}

// Provider represents the Databricks provider that registers MCP tools
type Provider struct {
	config  *mcp.Config
	session *session.Session
	ctx     context.Context
}

// NewProvider creates a new Databricks provider
func NewProvider(ctx context.Context, cfg *mcp.Config, sess *session.Session) (*Provider, error) {
	return &Provider{
		config:  cfg,
		session: sess,
		ctx:     ctx,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "databricks"
}

// RegisterTools registers all Databricks tools with the MCP server
func (p *Provider) RegisterTools(server *mcpsdk.Server) error {
	log.Info(p.ctx, "Registering Databricks tools")

	// Register databricks_list_catalogs
	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "databricks_list_catalogs",
			Description: "List all available Databricks catalogs",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args struct{}) (*mcpsdk.CallToolResult, any, error) {
			log.Debug(ctx, "databricks_list_catalogs called")

			result, err := ListCatalogs(ctx, p.config)
			if err != nil {
				return nil, nil, err
			}

			text := formatCatalogsResult(result)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: text},
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

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "databricks_list_schemas",
			Description: "List all schemas in a Databricks catalog with pagination support",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args ListSchemasInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "databricks_list_schemas called: catalog=%s", args.CatalogName)

			listArgs := &ListSchemasArgs{
				CatalogName: args.CatalogName,
				Filter:      args.Filter,
				Limit:       args.Limit,
				Offset:      args.Offset,
			}

			result, err := ListSchemas(ctx, p.config, listArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatSchemasResult(result)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: text},
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

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "databricks_list_tables",
			Description: "List tables in a Databricks catalog and schema",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args ListTablesInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "databricks_list_tables called: catalog=%s, schema=%s", args.CatalogName, args.SchemaName)

			listArgs := &ListTablesArgs{
				CatalogName:         args.CatalogName,
				SchemaName:          args.SchemaName,
				ExcludeInaccessible: args.ExcludeInaccessible,
			}

			result, err := ListTables(ctx, p.config, listArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatTablesResult(result)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register databricks_describe_table
	type DescribeTableInput struct {
		TableFullName string `json:"table_full_name" jsonschema:"required" jsonschema_description:"Full name of the table (catalog.schema.table)"`
		SampleSize    int    `json:"sample_size,omitempty" jsonschema_description:"Number of sample rows to return (default: 5)"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "databricks_describe_table",
			Description: "Get detailed information about a Databricks table including schema and optional sample data",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args DescribeTableInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "databricks_describe_table called: table=%s", args.TableFullName)

			descArgs := &DescribeTableArgs{
				TableFullName: args.TableFullName,
				SampleSize:    args.SampleSize,
			}

			result, err := DescribeTable(ctx, p.config, descArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatTableDetails(result)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register databricks_execute_query
	type ExecuteQueryInput struct {
		Query string `json:"query" jsonschema:"required" jsonschema_description:"SQL query to execute"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "databricks_execute_query",
			Description: "Execute SQL query in Databricks. Only single SQL statements are supported - do not send multiple statements separated by semicolons. For multiple statements, call this tool separately for each one. DO NOT create catalogs, schemas or tables - requires metastore admin privileges. Query existing data instead. Timeout: 60 seconds for query execution.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args ExecuteQueryInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "databricks_execute_query called: query=%s", args.Query)

			result, err := ExecuteQuery(ctx, p.config, args.Query)
			if err != nil {
				return nil, nil, err
			}

			text := formatQueryResult(result)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	log.Info(p.ctx, "Registered Databricks tools")
	return nil
}
