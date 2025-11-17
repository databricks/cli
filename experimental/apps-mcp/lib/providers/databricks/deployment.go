package databricks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type Status struct {
	Message string `json:"message"`
	State   string `json:"state"`
}

type DeploymentArtifacts struct {
	SourceCodePath string `json:"source_code_path"`
}

type Deployment struct {
	CreateTime          string              `json:"create_time"`
	Creator             string              `json:"creator"`
	DeploymentArtifacts DeploymentArtifacts `json:"deployment_artifacts"`
	DeploymentID        string              `json:"deployment_id"`
	Mode                string              `json:"mode"`
	SourceCodePath      string              `json:"source_code_path"`
	Status              Status              `json:"status"`
	UpdateTime          string              `json:"update_time"`
}

type AppInfo struct {
	ActiveDeployment         *Deployment `json:"active_deployment,omitempty"`
	AppStatus                Status      `json:"app_status"`
	ComputeStatus            Status      `json:"compute_status"`
	CreateTime               string      `json:"create_time"`
	Creator                  string      `json:"creator"`
	DefaultSourceCodePath    string      `json:"default_source_code_path"`
	Description              string      `json:"description"`
	EffectiveBudgetPolicyID  string      `json:"effective_budget_policy_id"`
	ID                       string      `json:"id"`
	Name                     string      `json:"name"`
	OAuth2AppClientID        string      `json:"oauth2_app_client_id"`
	OAuth2AppIntegrationID   string      `json:"oauth2_app_integration_id"`
	ServicePrincipalClientID string      `json:"service_principal_client_id"`
	ServicePrincipalID       int64       `json:"service_principal_id"`
	ServicePrincipalName     string      `json:"service_principal_name"`
	UpdateTime               string      `json:"update_time"`
	Updater                  string      `json:"updater"`
	URL                      string      `json:"url"`
}

func (a *AppInfo) SourcePath() string {
	if a.DefaultSourceCodePath == "" {
		return fmt.Sprintf("/Workspace/Users/%s/%s/", a.Creator, a.Name)
	}
	return a.DefaultSourceCodePath
}

type Permission string

const (
	PermissionCanUse    Permission = "CAN_USE"
	PermissionCanManage Permission = "CAN_MANAGE"
)

type Warehouse struct {
	ID         string     `json:"id"`
	Permission Permission `json:"permission"`
}

type Resources struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	SQLWarehouse *Warehouse `json:"sql_warehouse,omitempty"`
}

type CreateAppRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Resources   []Resources `json:"resources,omitempty"`
}

type UserInfo struct {
	ID          string `json:"id"`
	Active      bool   `json:"active"`
	DisplayName string `json:"displayName"`
	UserName    string `json:"userName"`
}

func GetAppInfo(ctx context.Context, cfg *mcp.Config, name string) (*AppInfo, error) {
	w := cmdctx.WorkspaceClient(ctx)
	app, err := w.Apps.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get app info: %w", err)
	}

	return convertAppToAppInfo(app), nil
}

func convertAppToAppInfo(app *apps.App) *AppInfo {
	appInfo := &AppInfo{
		AppStatus: Status{
			Message: app.AppStatus.Message,
			State:   string(app.AppStatus.State),
		},
		ComputeStatus: Status{
			Message: app.ComputeStatus.Message,
			State:   string(app.ComputeStatus.State),
		},
		CreateTime:               app.CreateTime,
		Creator:                  app.Creator,
		DefaultSourceCodePath:    app.DefaultSourceCodePath,
		Description:              app.Description,
		EffectiveBudgetPolicyID:  app.EffectiveBudgetPolicyId,
		ID:                       app.Id,
		Name:                     app.Name,
		OAuth2AppClientID:        app.Oauth2AppClientId,
		OAuth2AppIntegrationID:   app.Oauth2AppIntegrationId,
		ServicePrincipalClientID: app.ServicePrincipalClientId,
		ServicePrincipalID:       app.ServicePrincipalId,
		ServicePrincipalName:     app.ServicePrincipalName,
		UpdateTime:               app.UpdateTime,
		Updater:                  app.Updater,
		URL:                      app.Url,
	}

	if app.ActiveDeployment != nil {
		appInfo.ActiveDeployment = convertAppDeploymentToDeployment(app.ActiveDeployment)
	}

	return appInfo
}

func convertAppDeploymentToDeployment(dep *apps.AppDeployment) *Deployment {
	deployment := &Deployment{
		CreateTime:     dep.CreateTime,
		Creator:        dep.Creator,
		DeploymentID:   dep.DeploymentId,
		Mode:           string(dep.Mode),
		SourceCodePath: dep.SourceCodePath,
		UpdateTime:     dep.UpdateTime,
	}

	if dep.DeploymentArtifacts != nil {
		deployment.DeploymentArtifacts = DeploymentArtifacts{
			SourceCodePath: dep.DeploymentArtifacts.SourceCodePath,
		}
	}

	if dep.Status != nil {
		deployment.Status = Status{
			Message: dep.Status.Message,
			State:   string(dep.Status.State),
		}
	}

	return deployment
}

func CreateApp(ctx context.Context, cfg *mcp.Config, app *CreateAppRequest) (*AppInfo, error) {
	w := cmdctx.WorkspaceClient(ctx)

	sdkApp := apps.App{
		Name:        app.Name,
		Description: app.Description,
	}

	if len(app.Resources) > 0 {
		sdkApp.Resources = make([]apps.AppResource, len(app.Resources))
		for i, res := range app.Resources {
			resource := apps.AppResource{
				Name:        res.Name,
				Description: res.Description,
			}

			if res.SQLWarehouse != nil {
				resource.SqlWarehouse = &apps.AppResourceSqlWarehouse{
					Id:         res.SQLWarehouse.ID,
					Permission: apps.AppResourceSqlWarehouseSqlWarehousePermission(res.SQLWarehouse.Permission),
				}
			}

			sdkApp.Resources[i] = resource
		}
	}

	req := apps.CreateAppRequest{
		App: sdkApp,
	}

	wait, err := w.Apps.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	createdApp, err := wait.GetWithTimeout(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for app creation: %w", err)
	}

	return convertAppToAppInfo(createdApp), nil
}

func GetUserInfo(ctx context.Context, cfg *mcp.Config) (*UserInfo, error) {
	w := cmdctx.WorkspaceClient(ctx)
	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return convertUserToUserInfo(user), nil
}

func convertUserToUserInfo(user *iam.User) *UserInfo {
	return &UserInfo{
		ID:          user.Id,
		Active:      user.Active,
		DisplayName: user.DisplayName,
		UserName:    user.UserName,
	}
}

func SyncWorkspace(appInfo *AppInfo, sourceDir string) error {
	targetPath := appInfo.SourcePath()

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

func DeployApp(ctx context.Context, cfg *mcp.Config, appInfo *AppInfo) error {
	w := cmdctx.WorkspaceClient(ctx)
	sourcePath := appInfo.SourcePath()

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

func ResourcesFromEnv() (*Resources, error) {
	warehouseID := os.Getenv("DATABRICKS_WAREHOUSE_ID")
	if warehouseID == "" {
		return nil, errors.New("DATABRICKS_WAREHOUSE_ID environment variable is required for app deployment. Set this to your Databricks SQL warehouse ID")
	}

	return &Resources{
		Name:        "base",
		Description: "template resources",
		SQLWarehouse: &Warehouse{
			ID:         warehouseID,
			Permission: PermissionCanUse,
		},
	}, nil
}
