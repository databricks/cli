package io

import (
	"context"
	"log/slog"

	"github.com/databricks/cli/internal/mcp/templates"
	"github.com/databricks/cli/libs/mcp/config"
	"github.com/databricks/cli/libs/mcp/providers"
	"github.com/databricks/cli/libs/mcp/session"
	pkgtemplates "github.com/databricks/cli/libs/mcp/templates"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	providers.Register("io", func(cfg *config.Config, sess *session.Session, logger *slog.Logger) (providers.Provider, error) {
		return NewProvider(cfg.IoConfig, sess, logger)
	}, providers.ProviderConfig{
		Always: true,
	})
}

// Provider implements the I/O provider for scaffolding and validation
type Provider struct {
	config          *config.IoConfig
	session         *session.Session
	logger          *slog.Logger
	defaultTemplate pkgtemplates.Template
}

// NewProvider creates a new I/O provider
func NewProvider(cfg *config.IoConfig, sess *session.Session, logger *slog.Logger) (*Provider, error) {
	return &Provider{
		config:          cfg,
		session:         sess,
		logger:          logger,
		defaultTemplate: templates.GetTRPCTemplate(),
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "io"
}

// RegisterTools registers all I/O tools with the MCP server
func (p *Provider) RegisterTools(server *mcp.Server) error {
	p.logger.Info("Registering I/O tools")

	// Register scaffold_data_app
	type ScaffoldInput struct {
		WorkDir      string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory"`
		ForceRewrite bool   `json:"force_rewrite,omitempty" jsonschema_description:"Overwrite existing files if directory is not empty"`
	}

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "scaffold_data_app",
			Description: "Initialize a project by copying template files from the default TypeScript (tRPC + React) template to a work directory. Supports force rewrite to wipe and recreate the directory. It sets up a basic project structure, and should be ALWAYS used as the first step in creating a new data or web app.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args ScaffoldInput) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("scaffold_data_app called", "work_dir", args.WorkDir)

			scaffoldArgs := &ScaffoldArgs{
				WorkDir:      args.WorkDir,
				ForceRewrite: args.ForceRewrite,
			}

			result, err := p.Scaffold(ctx, scaffoldArgs)
			if err != nil {
				return nil, nil, err
			}

			// Set work directory in session for workspace tools
			if err := p.session.SetWorkDir(result.WorkDir); err != nil {
				p.logger.Warn("Failed to set work directory in session", "error", err)
			} else {
				p.logger.Info("Work directory set in session", "work_dir", result.WorkDir)
			}

			text := formatScaffoldResult(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	// Register validate_data_app
	type ValidateInput struct {
		WorkDir string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory"`
	}

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "validate_data_app",
			Description: "Validate a project by copying files to a sandbox and running validation checks. Project should be scaffolded first. Returns validation result with success status and details.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args ValidateInput) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("validate_data_app called", "work_dir", args.WorkDir)

			validateArgs := &ValidateArgs{
				WorkDir: args.WorkDir,
			}

			result, err := p.Validate(ctx, validateArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatValidateResult(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
				IsError: !result.Success,
			}, nil, nil
		}),
	)

	p.logger.Info("Registered I/O tools", "count", 2)
	return nil
}
