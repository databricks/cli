package deployment

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/mcp/config"
	"github.com/databricks/cli/libs/mcp/providers"
	"github.com/databricks/cli/libs/mcp/providers/databricks"
	"github.com/databricks/cli/libs/mcp/providers/io"
	"github.com/databricks/cli/libs/mcp/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	// Register deployment provider with conditional enablement based on AllowDeployment
	providers.Register("deployment", func(cfg *config.Config, sess *session.Session, logger *slog.Logger) (providers.Provider, error) {
		return NewProvider(cfg, sess, logger)
	}, providers.ProviderConfig{
		EnabledWhen: func(cfg *config.Config) bool {
			return cfg.AllowDeployment
		},
	})
}

const deployRetries = 3

// Provider implements Databricks app deployment functionality.
type Provider struct {
	config  *config.Config
	session *session.Session
	client  *databricks.Client
	logger  *slog.Logger
}

// DeployDatabricksAppInput contains parameters for deploying a Databricks app.
type DeployDatabricksAppInput struct {
	WorkDir     string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory containing the app to deploy"`
	Name        string `json:"name" jsonschema:"required" jsonschema_description:"Name of the Databricks app (alphanumeric and dash characters only)"`
	Description string `json:"description" jsonschema:"required" jsonschema_description:"Description of the Databricks app"`
	Force       bool   `json:"force,omitempty" jsonschema_description:"Force re-deployment if the app already exists"`
}

func NewProvider(cfg *config.Config, sess *session.Session, logger *slog.Logger) (*Provider, error) {
	client, err := databricks.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create databricks client: %w", err)
	}

	return &Provider{
		config:  cfg,
		session: sess,
		client:  client,
		logger:  logger,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "deployment"
}

func (p *Provider) RegisterTools(server *mcp.Server) error {
	p.logger.Info("Registering deployment tools")

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "deploy_databricks_app",
			Description: "Deploy a generated app to Databricks Apps. Creates the app if it doesn't exist, syncs local files to workspace, and deploys the app. Returns deployment status and app URL. Only use after direct user request and running validation.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcp.CallToolRequest, args DeployDatabricksAppInput) (*mcp.CallToolResult, any, error) {
			p.logger.Debug("deploy_databricks_app called",
				"work_dir", args.WorkDir,
				"name", args.Name,
				"force", args.Force,
			)

			if !filepath.IsAbs(args.WorkDir) {
				return nil, nil, fmt.Errorf("work_dir must be an absolute path, got: '%s'. Relative paths are not supported", args.WorkDir)
			}

			result, err := p.deployDatabricksApp(ctx, &args)
			if err != nil {
				return nil, nil, err
			}

			if !result.Success {
				return nil, nil, fmt.Errorf("%s", result.Message)
			}

			text := formatDeployResult(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		}),
	)

	return nil
}

// DeployResult contains the outcome of a Databricks app deployment.
type DeployResult struct {
	Success bool
	Message string
	AppURL  string
	AppName string
}

func (p *Provider) deployDatabricksApp(ctx context.Context, args *DeployDatabricksAppInput) (*DeployResult, error) {
	startTime := time.Now()

	workPath := args.WorkDir
	if _, err := os.Stat(workPath); os.IsNotExist(err) {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Work directory does not exist: %s", workPath),
			AppName: args.Name,
		}, nil
	}

	fileInfo, err := os.Stat(workPath)
	if err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to stat work directory: %v", err),
			AppName: args.Name,
		}, nil
	}

	if !fileInfo.IsDir() {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Work path is not a directory: %s", workPath),
			AppName: args.Name,
		}, nil
	}

	projectState, err := io.LoadState(workPath)
	if err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to load project state: %v", err),
			AppName: args.Name,
		}, nil
	}

	if projectState == nil {
		return &DeployResult{
			Success: false,
			Message: "Project must be scaffolded before deployment",
			AppName: args.Name,
		}, nil
	}

	expectedChecksum, hasChecksum := projectState.Checksum()
	if !hasChecksum {
		return &DeployResult{
			Success: false,
			Message: "Project must be validated before deployment. Run validate_data_app first.",
			AppName: args.Name,
		}, nil
	}

	checksumValid, err := io.VerifyChecksum(workPath, expectedChecksum)
	if err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to verify project checksum: %v", err),
			AppName: args.Name,
		}, nil
	}

	if !checksumValid {
		return &DeployResult{
			Success: false,
			Message: "Project files changed since validation. Re-run validate_data_app before deployment.",
			AppName: args.Name,
		}, nil
	}

	p.logger.Info("Installing dependencies", "work_dir", workPath)
	if err := p.runCommand(workPath, "npm", "install"); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to install dependencies: %v", err),
			AppName: args.Name,
		}, nil
	}

	p.logger.Info("Building frontend", "work_dir", workPath)
	if err := p.runCommand(workPath, "npm", "run", "build"); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to build frontend: %v", err),
			AppName: args.Name,
		}, nil
	}

	appInfo, err := p.getOrCreateApp(ctx, args.Name, args.Description, args.Force)
	if err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get or create app: %v", err),
			AppName: args.Name,
		}, nil
	}

	serverDir := filepath.Join(workPath, "server")
	syncStart := time.Now()
	p.logger.Info("Syncing workspace", "source", serverDir, "target", appInfo.SourcePath())

	if err := databricks.SyncWorkspace(appInfo, serverDir); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to sync workspace: %v", err),
			AppName: args.Name,
		}, nil
	}

	p.logger.Info("Workspace sync completed", "duration_seconds", time.Since(syncStart).Seconds())

	deployStart := time.Now()
	p.logger.Info("Deploying app", "name", args.Name)

	var deployErr error
	for attempt := 1; attempt <= deployRetries; attempt++ {
		deployErr = databricks.DeployApp(ctx, p.client, appInfo)
		if deployErr == nil {
			break
		}

		if attempt < deployRetries {
			p.logger.Warn("Deploy attempt failed, retrying",
				"attempt", attempt,
				"error", deployErr.Error(),
			)
		}
	}

	if deployErr != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to deploy app after %d attempts: %v", deployRetries, deployErr),
			AppName: args.Name,
		}, nil
	}

	p.logger.Info("App deployment completed", "duration_seconds", time.Since(deployStart).Seconds())

	deployedState, err := projectState.Deploy()
	if err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to transition state: %v", err),
			AppName: args.Name,
		}, nil
	}

	if err := io.SaveState(workPath, deployedState); err != nil {
		p.logger.Warn("Failed to save deployed state", "error", err)
	}

	totalDuration := time.Since(startTime)
	p.logger.Info("Full deployment completed",
		"duration_seconds", totalDuration.Seconds(),
		"app_url", appInfo.URL,
	)

	return &DeployResult{
		Success: true,
		Message: "Deployment completed successfully",
		AppURL:  appInfo.URL,
		AppName: args.Name,
	}, nil
}

func (p *Provider) getOrCreateApp(ctx context.Context, name, description string, force bool) (*databricks.AppInfo, error) {
	appInfo, err := databricks.GetAppInfo(ctx, p.client, name)
	if err == nil {
		p.logger.Info("Found existing app", "name", name)

		if !force {
			userInfo, err := databricks.GetUserInfo(ctx, p.client)
			if err != nil {
				return nil, fmt.Errorf("failed to get user info: %w", err)
			}

			if appInfo.Creator != userInfo.UserName {
				return nil, fmt.Errorf(
					"app '%s' already exists and was created by another user: %s. Use 'force' option to override",
					name,
					appInfo.Creator,
				)
			}
		}

		return appInfo, nil
	}

	p.logger.Info("App not found, creating new app", "name", name)

	resources, err := databricks.ResourcesFromEnv()
	if err != nil {
		return nil, err
	}

	createApp := &databricks.CreateAppRequest{
		Name:        name,
		Description: description,
		Resources:   []databricks.Resources{*resources},
	}

	return databricks.CreateApp(ctx, p.client, createApp)
}

func (p *Provider) runCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w (output: %s)", name, err, string(output))
	}

	return nil
}

func formatDeployResult(result *DeployResult) string {
	if result.Success {
		return fmt.Sprintf(
			"Successfully deployed app '%s'\nURL: %s\n%s",
			result.AppName,
			result.AppURL,
			result.Message,
		)
	}
	return fmt.Sprintf(
		"Deployment failed for app '%s': %s",
		result.AppName,
		result.Message,
	)
}
