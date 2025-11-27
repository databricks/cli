package deployment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	mcpsdk "github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers/databricks"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers/io"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

func init() {
	// Register deployment provider
	providers.Register("deployment", func(ctx context.Context, cfg *mcp.Config, sess *session.Session) (providers.Provider, error) {
		return NewProvider(ctx, cfg, sess)
	}, providers.ProviderConfig{})
}

const deployRetries = 3

// Provider implements Databricks app deployment functionality.
type Provider struct {
	config  *mcp.Config
	session *session.Session
	ctx     context.Context
}

// DeployDatabricksAppInput contains parameters for deploying a Databricks app.
type DeployDatabricksAppInput struct {
	WorkDir     string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory containing the app to deploy"`
	Name        string `json:"name" jsonschema:"required" jsonschema_description:"Name of the Databricks app (alphanumeric and dash characters only)"`
	Description string `json:"description" jsonschema:"required" jsonschema_description:"Description of the Databricks app"`
	Force       bool   `json:"force,omitempty" jsonschema_description:"Force re-deployment if the app already exists"`
}

func NewProvider(ctx context.Context, cfg *mcp.Config, sess *session.Session) (*Provider, error) {
	return &Provider{
		config:  cfg,
		session: sess,
		ctx:     ctx,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "deployment"
}

func (p *Provider) RegisterTools(server *mcpsdk.Server) error {
	log.Info(p.ctx, "Registering deployment tools")

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "deploy_databricks_app",
			Description: "Deploy a generated app to Databricks Apps. Creates the app if it doesn't exist, syncs local files to workspace, and deploys the app. Returns deployment status and app URL. Only use after direct user request and running validation.",
		},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, args DeployDatabricksAppInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "deploy_databricks_app called: work_dir=%s, name=%s, force=%v",
				args.WorkDir, args.Name, args.Force)

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
			return mcpsdk.CreateNewTextContentResult(text), nil, nil
		},
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
			Message: "Work directory does not exist: " + workPath,
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
			Message: "Work path is not a directory: " + workPath,
			AppName: args.Name,
		}, nil
	}

	projectState, err := io.LoadState(ctx, workPath)
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
			Message: "Project must be validated before deployment. Run validate_databricks_app first.",
			AppName: args.Name,
		}, nil
	}

	checksumValid, err := io.VerifyChecksum(ctx, workPath, expectedChecksum)
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
			Message: "Project files changed since validation. Re-run validate_databricks_app before deployment.",
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

	syncStart := time.Now()
	log.Infof(ctx, "Syncing workspace: source=%s, target=%s", workPath, databricks.GetSourcePath(appInfo))

	if err := databricks.SyncWorkspace(ctx, appInfo, workPath); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to sync workspace: %v", err),
			AppName: args.Name,
		}, nil
	}

	log.Infof(ctx, "Workspace sync completed: duration_seconds=%.2f", time.Since(syncStart).Seconds())

	deployStart := time.Now()
	log.Infof(ctx, "Deploying app: name=%s", args.Name)

	var deployErr error
	for attempt := 1; attempt <= deployRetries; attempt++ {
		deployErr = databricks.DeployApp(ctx, p.config, appInfo)
		if deployErr == nil {
			break
		}

		if attempt < deployRetries {
			log.Warnf(ctx, "Deploy attempt failed, retrying: attempt=%d, error=%s",
				attempt, deployErr.Error())
		}
	}

	if deployErr != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to deploy app after %d attempts: %v", deployRetries, deployErr),
			AppName: args.Name,
		}, nil
	}

	log.Infof(ctx, "App deployment completed: duration_seconds=%.2f", time.Since(deployStart).Seconds())

	deployedState, err := projectState.Deploy()
	if err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("Failed to transition state: %v", err),
			AppName: args.Name,
		}, nil
	}

	if err := io.SaveState(ctx, workPath, deployedState); err != nil {
		log.Warnf(ctx, "Failed to save deployed state: error=%v", err)
	}

	totalDuration := time.Since(startTime)
	log.Infof(ctx, "Full deployment completed: duration_seconds=%.2f, app_url=%s",
		totalDuration.Seconds(), appInfo.Url)

	return &DeployResult{
		Success: true,
		Message: "Deployment completed successfully",
		AppURL:  appInfo.Url,
		AppName: args.Name,
	}, nil
}

func (p *Provider) getOrCreateApp(ctx context.Context, name, description string, force bool) (*apps.App, error) {
	appInfo, err := databricks.GetAppInfo(ctx, name)
	if err == nil {
		log.Infof(ctx, "Found existing app: name=%s", name)

		if !force {
			userInfo, err := databricks.GetUserInfo(ctx, p.config)
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

	log.Infof(ctx, "App not found, creating new app: name=%s", name)

	resources, err := databricks.ResourcesFromEnv(ctx, p.config)
	if err != nil {
		return nil, err
	}

	createApp := &apps.CreateAppRequest{
		App: apps.App{
			Name:        name,
			Description: description,
			Resources:   []apps.AppResource{*resources},
		},
	}

	return databricks.CreateApp(ctx, createApp)
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
