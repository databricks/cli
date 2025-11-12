package io

import (
	"context"

	"github.com/databricks/cli/internal/mcp/templates"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/providers"
	"github.com/databricks/cli/libs/mcp/session"
	pkgtemplates "github.com/databricks/cli/libs/mcp/templates"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	providers.Register("io", func(cfg *mcp.Config, sess *session.Session, ctx context.Context) (providers.Provider, error) {
		return NewProvider(cfg.IoConfig, sess, ctx)
	}, providers.ProviderConfig{
		Always: true,
	})
}

// Provider implements the I/O provider for scaffolding and validation
type Provider struct {
	config          *mcp.IoConfig
	session         *session.Session
	ctx             context.Context
	defaultTemplate pkgtemplates.Template
}

// NewProvider creates a new I/O provider
func NewProvider(cfg *mcp.IoConfig, sess *session.Session, ctx context.Context) (*Provider, error) {
	return &Provider{
		config:          cfg,
		session:         sess,
		ctx:             ctx,
		defaultTemplate: templates.GetTRPCTemplate(),
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "io"
}

// RegisterTools registers all I/O tools with the MCP server
func (p *Provider) RegisterTools(server *mcpsdk.Server) error {
	log.Info(p.ctx, "Registering I/O tools")

	// Register scaffold_data_app
	type ScaffoldInput struct {
		WorkDir      string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory"`
		ForceRewrite bool   `json:"force_rewrite,omitempty" jsonschema_description:"Overwrite existing files if directory is not empty"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "scaffold_data_app",
			Description: "Initialize a project by copying template files from the default TypeScript (tRPC + React) template to a work directory. Supports force rewrite to wipe and recreate the directory. It sets up a basic project structure, and should be ALWAYS used as the first step in creating a new data or web app.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args ScaffoldInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "scaffold_data_app called: work_dir=%s", args.WorkDir)

			scaffoldArgs := &ScaffoldArgs{
				WorkDir:      args.WorkDir,
				ForceRewrite: args.ForceRewrite,
			}

			result, err := p.Scaffold(ctx, scaffoldArgs)
			if err != nil {
				return nil, nil, err
			}

			// Set work directory in session for workspace tools
			if err := session.SetWorkDir(ctx, result.WorkDir); err != nil {
				log.Warnf(ctx, "Failed to set work directory in session: error=%v", err)
			} else {
				log.Infof(ctx, "Work directory set in session: work_dir=%s", result.WorkDir)
			}

			text := formatScaffoldResult(result)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register validate_data_app
	type ValidateInput struct {
		WorkDir string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "validate_data_app",
			Description: "Validate a project by copying files to a sandbox and running validation checks. Project should be scaffolded first. Returns validation result with success status and details.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args ValidateInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "validate_data_app called: work_dir=%s", args.WorkDir)

			validateArgs := &ValidateArgs{
				WorkDir: args.WorkDir,
			}

			result, err := p.Validate(ctx, validateArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatValidateResult(result)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: text},
				},
				IsError: !result.Success,
			}, nil, nil
		}),
	)

	log.Infof(p.ctx, "Registered I/O tools: count=%d", 2)
	return nil
}
