# Phase 1: Create Databricks Deployment Client Package

## Objective
Create a Go package that wraps the Databricks CLI for deployment operations in `pkg/providers/databricks/deployment.go`.

## Context
The Rust implementation at `edda_mcp/src/providers/deployment.rs` and `edda_integrations/src/deployment.rs` provides the reference implementation. The Go version should mirror this functionality while following Go idioms.

## Prerequisites
- Databricks CLI must be installed and configured on the system
- The CLI is invoked via `os/exec` package
- All CLI commands return JSON that needs to be parsed

## Implementation Details

### File Location
Create: `go-mcp/pkg/providers/databricks/deployment.go`

### Data Structures

Implement the following structs that mirror the Rust types in `edda_integrations/src/deployment.rs`:

```go
package databricks

import (
    "encoding/json"
    "fmt"
    "os/exec"
)

// Status represents app/deployment status
type Status struct {
    Message string `json:"message"`
    State   string `json:"state"`
}

// DeploymentArtifacts contains source code path info
type DeploymentArtifacts struct {
    SourceCodePath string `json:"source_code_path"`
}

// Deployment represents a Databricks app deployment
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

// AppInfo represents complete Databricks app information
type AppInfo struct {
    ActiveDeployment            *Deployment `json:"active_deployment,omitempty"`
    AppStatus                   Status      `json:"app_status"`
    ComputeStatus               Status      `json:"compute_status"`
    CreateTime                  string      `json:"create_time"`
    Creator                     string      `json:"creator"`
    DefaultSourceCodePath       string      `json:"default_source_code_path"`
    Description                 string      `json:"description"`
    EffectiveBudgetPolicyID     string      `json:"effective_budget_policy_id"`
    ID                          string      `json:"id"`
    Name                        string      `json:"name"`
    OAuth2AppClientID           string      `json:"oauth2_app_client_id"`
    OAuth2AppIntegrationID      string      `json:"oauth2_app_integration_id"`
    ServicePrincipalClientID    string      `json:"service_principal_client_id"`
    ServicePrincipalID          int64       `json:"service_principal_id"`
    ServicePrincipalName        string      `json:"service_principal_name"`
    UpdateTime                  string      `json:"update_time"`
    Updater                     string      `json:"updater"`
    URL                         string      `json:"url"`
}

// SourcePath returns the workspace path for the app
func (a *AppInfo) SourcePath() string {
    if a.DefaultSourceCodePath == "" {
        return fmt.Sprintf("/Workspace/Users/%s/%s/", a.Creator, a.Name)
    }
    return a.DefaultSourceCodePath
}

// Permission represents warehouse permission level
type Permission string

const (
    PermissionCanUse   Permission = "CAN_USE"
    PermissionCanManage Permission = "CAN_MANAGE"
)

// Warehouse represents SQL warehouse configuration
type Warehouse struct {
    ID         string     `json:"id"`
    Permission Permission `json:"permission"`
}

// Resources represents app resources (SQL warehouse, etc.)
type Resources struct {
    Name         string     `json:"name"`
    Description  string     `json:"description"`
    SQLWarehouse *Warehouse `json:"sql_warehouse,omitempty"`
}

// CreateApp represents app creation parameters
type CreateApp struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Resources   []Resources `json:"resources,omitempty"`
}

// UserInfo represents current Databricks user
type UserInfo struct {
    ID          string `json:"id"`
    Active      bool   `json:"active"`
    DisplayName string `json:"displayName"`
    UserName    string `json:"userName"`
}
```

### Core Functions

Implement these functions following the Rust implementations:

#### 1. GetAppInfo
```go
// GetAppInfo retrieves app information from Databricks
// Executes: databricks apps get <name>
// Reference: edda_integrations/src/deployment.rs:155-170
func GetAppInfo(name string) (*AppInfo, error) {
    cmd := exec.Command("databricks", "apps", "get", name)
    output, err := cmd.Output()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            return nil, fmt.Errorf("failed to get app info: %s", string(exitErr.Stderr))
        }
        return nil, fmt.Errorf("failed to execute databricks CLI: %w", err)
    }

    var appInfo AppInfo
    if err := json.Unmarshal(output, &appInfo); err != nil {
        return nil, fmt.Errorf("failed to parse app info: %w", err)
    }

    return &appInfo, nil
}
```

#### 2. CreateApp
```go
// CreateApp creates a new Databricks app
// Executes: databricks apps create --json <json>
// Reference: edda_integrations/src/deployment.rs:172-186
func CreateApp(app *CreateApp) (*AppInfo, error) {
    jsonData, err := json.Marshal(app)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal create app request: %w", err)
    }

    cmd := exec.Command("databricks", "apps", "create", "--json", string(jsonData))
    output, err := cmd.Output()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            return nil, fmt.Errorf("failed to create app: %s", string(exitErr.Stderr))
        }
        return nil, fmt.Errorf("failed to execute databricks CLI: %w", err)
    }

    // After creation, get the full app info
    return GetAppInfo(app.Name)
}
```

#### 3. GetUserInfo
```go
// GetUserInfo retrieves current Databricks user information
// Executes: databricks current-user me
// Reference: edda_integrations/src/deployment.rs:233-247
func GetUserInfo() (*UserInfo, error) {
    cmd := exec.Command("databricks", "current-user", "me")
    output, err := cmd.Output()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            return nil, fmt.Errorf("failed to get user info: %s", string(exitErr.Stderr))
        }
        return nil, fmt.Errorf("failed to execute databricks CLI: %w", err)
    }

    var userInfo UserInfo
    if err := json.Unmarshal(output, &userInfo); err != nil {
        return nil, fmt.Errorf("failed to parse user info: %w", err)
    }

    return &userInfo, nil
}
```

#### 4. SyncWorkspace
```go
// SyncWorkspace syncs local directory to Databricks workspace
// Executes: databricks sync --include public --exclude node_modules . <target_path>
// Reference: edda_integrations/src/deployment.rs:188-210
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
        return fmt.Errorf("failed to sync workspace: %s", string(output))
    }

    return nil
}
```

#### 5. DeployApp
```go
// DeployApp deploys the app to Databricks
// Executes: databricks apps deploy <name> --source-code-path <path>
// Reference: edda_integrations/src/deployment.rs:212-231
func DeployApp(appInfo *AppInfo) error {
    sourcePath := appInfo.SourcePath()

    cmd := exec.Command(
        "databricks",
        "apps",
        "deploy",
        appInfo.Name,
        "--source-code-path",
        sourcePath,
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to deploy app: %s", string(output))
    }

    return nil
}
```

#### 6. ResourcesFromEnv
```go
// ResourcesFromEnv creates Resources from environment variables
// Reference: edda_integrations/src/deployment.rs:82-97
func ResourcesFromEnv() (*Resources, error) {
    warehouseID := os.Getenv("DATABRICKS_WAREHOUSE_ID")
    if warehouseID == "" {
        return nil, fmt.Errorf("DATABRICKS_WAREHOUSE_ID environment variable is required for app deployment. Set this to your Databricks SQL warehouse ID")
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
```

## Testing

Create basic unit tests in `pkg/providers/databricks/deployment_test.go`:

```go
package databricks

import (
    "encoding/json"
    "testing"
)

func TestAppInfoSourcePath(t *testing.T) {
    tests := []struct {
        name     string
        appInfo  AppInfo
        expected string
    }{
        {
            name: "default path empty",
            appInfo: AppInfo{
                Creator:               "user@example.com",
                Name:                  "my-app",
                DefaultSourceCodePath: "",
            },
            expected: "/Workspace/Users/user@example.com/my-app/",
        },
        {
            name: "custom path set",
            appInfo: AppInfo{
                Creator:               "user@example.com",
                Name:                  "my-app",
                DefaultSourceCodePath: "/custom/path/",
            },
            expected: "/custom/path/",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.appInfo.SourcePath()
            if result != tt.expected {
                t.Errorf("expected %s, got %s", tt.expected, result)
            }
        })
    }
}

func TestResourcesSerialization(t *testing.T) {
    resources := Resources{
        Name:        "base",
        Description: "template resources",
        SQLWarehouse: &Warehouse{
            ID:         "warehouse-123",
            Permission: PermissionCanUse,
        },
    }

    data, err := json.Marshal(resources)
    if err != nil {
        t.Fatalf("failed to marshal: %v", err)
    }

    var decoded Resources
    if err := json.Unmarshal(data, &decoded); err != nil {
        t.Fatalf("failed to unmarshal: %v", err)
    }

    if decoded.SQLWarehouse.ID != "warehouse-123" {
        t.Errorf("expected warehouse-123, got %s", decoded.SQLWarehouse.ID)
    }
}
```

## Verification

After implementation:
1. Run `go build ./pkg/providers/databricks`
2. Run `go test ./pkg/providers/databricks`
3. Verify all tests pass
4. Ensure no compilation errors

## Success Criteria
- [ ] All data structures defined
- [ ] All 6 core functions implemented
- [ ] Basic unit tests pass
- [ ] No compilation errors
- [ ] Code follows Go idioms and conventions
