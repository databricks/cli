package io

import (
	"context"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	mcpsdk "github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/experimental/apps-mcp/lib/templates"
	"github.com/databricks/cli/libs/log"
)

func init() {
	providers.Register("io", func(ctx context.Context, cfg *mcp.Config, sess *session.Session) (providers.Provider, error) {
		return NewProvider(ctx, cfg.IoConfig, sess)
	}, providers.ProviderConfig{
		Always: true,
	})
}

// Provider implements the I/O provider for scaffolding and validation
type Provider struct {
	config          *mcp.IoConfig
	session         *session.Session
	ctx             context.Context
	defaultTemplate templates.Template
}

// NewProvider creates a new I/O provider
func NewProvider(ctx context.Context, cfg *mcp.IoConfig, sess *session.Session) (*Provider, error) {
	return &Provider{
		config:          cfg,
		session:         sess,
		ctx:             ctx,
		defaultTemplate: templates.GetAppKitTemplate(),
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "io"
}

// RegisterTools registers all I/O tools with the MCP server
func (p *Provider) RegisterTools(server *mcpsdk.Server) error {
	log.Info(p.ctx, "Registering I/O tools")

	// Register scaffold_databricks_app
	type ScaffoldInput struct {
		WorkDir        string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory"`
		AppName        string `json:"app_name" jsonschema:"required" jsonschema_description:"Name of the app (alphanumeric and dash characters only)"`
		AppDescription string `json:"app_description,omitempty" jsonschema_description:"Description of the app (max 100 characters)"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "scaffold_databricks_app",
			Description: "MUST use this tool when creating a new (Databricks) web app. Initialize a project by copying template files from the default TypeScript (AppKit) template to a work directory. It sets up a basic project structure.",
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args ScaffoldInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "scaffold_databricks_app called: work_dir=%s", args.WorkDir)

			scaffoldArgs := &ScaffoldArgs{
				WorkDir:        args.WorkDir,
				AppName:        args.AppName,
				AppDescription: args.AppDescription,
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
			return mcpsdk.CreateNewTextContentResult(text), nil, nil
		},
	)

	// Register validate_databricks_app
	type ValidateInput struct {
		WorkDir string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "validate_databricks_app",
			Description: "Validate a project by running validation checks. Project should be scaffolded first. Returns validation result with success status and details.",
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args ValidateInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "validate_databricks_app called: work_dir=%s", args.WorkDir)

			validateArgs := &ValidateArgs{
				WorkDir: args.WorkDir,
			}

			result, err := p.Validate(ctx, validateArgs)
			if err != nil {
				return nil, nil, err
			}

			text := formatValidateResult(result)
			return mcpsdk.CreateNewTextContentResult(text), nil, nil
		},
	)

	log.Infof(p.ctx, "Registered I/O tools: count=%d", 2)
	return nil
}
