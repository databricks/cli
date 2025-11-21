package databricks

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

func GetSourcePath(app *apps.App) string {
	if app.DefaultSourceCodePath == "" {
		return fmt.Sprintf("/Workspace/Users/%s/%s/", app.Creator, app.Name)
	}
	return app.DefaultSourceCodePath
}

func GetAppInfo(ctx context.Context, cfg *mcp.Config, name string) (*apps.App, error) {
	w := cmdctx.WorkspaceClient(ctx)
	app, err := w.Apps.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get app info: %w", err)
	}

	return app, nil
}

func CreateApp(ctx context.Context, cfg *mcp.Config, createAppRequest *apps.CreateAppRequest) (*apps.App, error) {
	w := cmdctx.WorkspaceClient(ctx)

	wait, err := w.Apps.Create(ctx, *createAppRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	createdApp, err := wait.GetWithTimeout(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for app creation: %w", err)
	}

	return createdApp, nil
}

func GetUserInfo(ctx context.Context, cfg *mcp.Config) (*iam.User, error) {
	w := cmdctx.WorkspaceClient(ctx)
	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return user, nil
}

func SyncWorkspace(appInfo *apps.App, sourceDir string) error {
	targetPath := GetSourcePath(appInfo)

	cmd := exec.Command(
		"databricks",
		"sync",
		"--include", "public",
		"--exclude", "node_modules",
		".",
		targetPath,
	)
	cmd.Dir = sourceDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to sync workspace: %w (output: %s)", err, string(output))
	}

	return nil
}

func DeployApp(ctx context.Context, cfg *mcp.Config, appInfo *apps.App) error {
	w := cmdctx.WorkspaceClient(ctx)
	sourcePath := GetSourcePath(appInfo)

	req := apps.CreateAppDeploymentRequest{
		AppName: appInfo.Name,
		AppDeployment: apps.AppDeployment{
			SourceCodePath: sourcePath,
			Mode:           apps.AppDeploymentModeSnapshot,
		},
	}

	wait, err := w.Apps.Deploy(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to deploy app: %w", err)
	}

	_, err = wait.GetWithTimeout(10 * time.Minute)
	if err != nil {
		return fmt.Errorf("failed to wait for app deployment: %w", err)
	}

	return nil
}

func ResourcesFromEnv(cfg *mcp.Config) (*apps.AppResource, error) {
	warehouseID := cfg.WarehouseID

	return &apps.AppResource{
		Name:        "base",
		Description: "template resources",
		SqlWarehouse: &apps.AppResourceSqlWarehouse{
			Id:         warehouseID,
			Permission: apps.AppResourceSqlWarehouseSqlWarehousePermissionCanUse,
		},
	}, nil
}
