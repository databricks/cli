# Phase 2: Create Deployment Provider

## Objective
Create a new MCP provider that exposes the `deploy_databricks_app` tool, implementing the deployment workflow with state management, validation, and error handling.

## Context
This phase builds on Phase 1 (Databricks client) and integrates with existing Go infrastructure:
- State management: `pkg/providers/io/state.go`
- Session management: `pkg/session/session.go`
- Config: `pkg/config/config.go`

Reference implementation: `edda_mcp/src/providers/deployment.rs:17-267`

## Prerequisites
- Phase 1 completed (`pkg/providers/databricks/deployment.go` exists)
- Familiarity with existing providers: `pkg/providers/io/provider.go`, `pkg/providers/workspace/provider.go`

## Implementation Details

### File Location
Create: `go-mcp/pkg/providers/deployment/provider.go`

### Package Structure

```go
package deployment

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "time"

    "github.com/appdotbuild/go-mcp/pkg/config"
    "github.com/appdotbuild/go-mcp/pkg/providers/databricks"
    "github.com/appdotbuild/go-mcp/pkg/providers/io"
    "github.com/appdotbuild/go-mcp/pkg/session"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

const deployRetries = 3

// Provider implements the deployment provider for Databricks apps
type Provider struct {
    config  *config.Config
    session *session.Session
    logger  *slog.Logger
}

// NewProvider creates a new deployment provider
func NewProvider(cfg *config.Config, sess *session.Session, logger *slog.Logger) (*Provider, error) {
    return &Provider{
        config:  cfg,
        session: sess,
        logger:  logger,
    }, nil
}
```

### MCP Tool Registration

```go
// RegisterTools registers all deployment tools with the MCP server
func (p *Provider) RegisterTools(server *mcp.Server) error {
    p.logger.Info("Registering deployment tools")

    // Register deploy_databricks_app
    type DeployDatabricksAppInput struct {
        WorkDir     string `json:"work_dir" jsonschema:"required" jsonschema_description:"Absolute path to the work directory containing the app to deploy"`
        Name        string `json:"name" jsonschema:"required" jsonschema_description:"Name of the Databricks app (alphanumeric and dash characters only)"`
        Description string `json:"description" jsonschema:"required" jsonschema_description:"Description of the Databricks app"`
        Force       bool   `json:"force,omitempty" jsonschema_description:"Force re-deployment if the app already exists"`
    }

    mcp.AddTool(server,
        &mcp.Tool{
            Name:        "deploy_databricks_app",
            Description: "Deploy a generated app to Databricks Apps. Creates the app if it doesn't exist, syncs local files to workspace, and deploys the app. Returns deployment status and app URL. Only use after direct user request and running validation.",
        },
        func(ctx context.Context, req *mcp.CallToolRequest, args DeployDatabricksAppInput) (*mcp.CallToolResult, any, error) {
            p.logger.Debug("deploy_databricks_app called",
                "work_dir", args.WorkDir,
                "name", args.Name,
                "force", args.Force,
            )

            // Validate absolute path
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
        },
    )

    return nil
}
```

### Core Deployment Logic

```go
// DeployResult represents the deployment result
type DeployResult struct {
    Success bool
    Message string
    AppURL  string
    AppName string
}

// deployDatabricksApp implements the core deployment logic
// Reference: edda_mcp/src/providers/deployment.rs:69-227
func (p *Provider) deployDatabricksApp(ctx context.Context, args *DeployDatabricksAppInput) (*DeployResult, error) {
    startTime := time.Now()

    // Step 1: Validate work directory exists
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

    // Step 2: Load and verify state
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

    // Step 3: Verify checksum hasn't changed since validation
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

    // Step 4: Install dependencies
    p.logger.Info("Installing dependencies", "work_dir", workPath)
    if err := p.runCommand(workPath, "npm", "install"); err != nil {
        return &DeployResult{
            Success: false,
            Message: fmt.Sprintf("Failed to install dependencies: %v", err),
            AppName: args.Name,
        }, nil
    }

    // Step 5: Build frontend
    p.logger.Info("Building frontend", "work_dir", workPath)
    if err := p.runCommand(workPath, "npm", "run", "build"); err != nil {
        return &DeployResult{
            Success: false,
            Message: fmt.Sprintf("Failed to build frontend: %v", err),
            AppName: args.Name,
        }, nil
    }

    // Step 6: Get or create app
    appInfo, err := p.getOrCreateApp(args.Name, args.Description, args.Force)
    if err != nil {
        return &DeployResult{
            Success: false,
            Message: fmt.Sprintf("Failed to get or create app: %v", err),
            AppName: args.Name,
        }, nil
    }

    // Step 7: Sync workspace
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

    // Step 8: Deploy app with retries
    deployStart := time.Now()
    p.logger.Info("Deploying app", "name", args.Name)

    var deployErr error
    for attempt := 1; attempt <= deployRetries; attempt++ {
        deployErr = databricks.DeployApp(appInfo)
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

    // Step 9: Transition to deployed state
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

// getOrCreateApp gets existing app or creates new one
// Reference: edda_mcp/src/providers/deployment.rs:159-179
func (p *Provider) getOrCreateApp(name, description string, force bool) (*databricks.AppInfo, error) {
    // Try to get existing app
    appInfo, err := databricks.GetAppInfo(name)
    if err == nil {
        p.logger.Info("Found existing app", "name", name)

        // Check ownership if not forcing
        if !force {
            userInfo, err := databricks.GetUserInfo()
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

    // App doesn't exist, create it
    p.logger.Info("App not found, creating new app", "name", name)

    resources, err := databricks.ResourcesFromEnv()
    if err != nil {
        return nil, err
    }

    createApp := &databricks.CreateApp{
        Name:        name,
        Description: description,
        Resources:   []databricks.Resources{*resources},
    }

    return databricks.CreateApp(createApp)
}

// runCommand executes a command in the specified directory
func (p *Provider) runCommand(dir string, name string, args ...string) error {
    cmd := exec.Command(name, args...)
    cmd.Dir = dir

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("%s failed: %s", name, string(output))
    }

    return nil
}

// formatDeployResult formats the deployment result for display
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
```

## Verification

After implementation:
1. Run `go build ./pkg/providers/deployment`
2. Ensure no compilation errors
3. Verify integration with existing packages

## Success Criteria
- [ ] Provider struct created with NewProvider constructor
- [ ] RegisterTools method implemented
- [ ] deploy_databricks_app tool registered with proper schema
- [ ] Core deployment logic implemented with all 9 steps
- [ ] Ownership checks implemented
- [ ] Retry logic for deployment (3 attempts)
- [ ] State transitions handled correctly
- [ ] Progress logging with durations
- [ ] No compilation errors
