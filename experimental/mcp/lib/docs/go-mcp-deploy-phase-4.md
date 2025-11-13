# Phase 4: Testing

## Objective
Create comprehensive unit tests and integration tests for the deployment functionality, ensuring reliability and correctness.

## Context
Testing strategy follows Go best practices with table-driven tests. Integration tests require Databricks credentials and are optional (can be skipped in CI).

Reference Rust tests: `edda_integrations/src/deployment.rs:249-269`

## Prerequisites
- Phase 1 completed (`pkg/providers/databricks/deployment.go`)
- Phase 2 completed (`pkg/providers/deployment/provider.go`)
- Phase 3 completed (integration in CLI)

## Implementation Details

### Test File 1: Databricks Client Tests

Create: `go-mcp/pkg/providers/databricks/deployment_test.go`

```go
package databricks

import (
    "encoding/json"
    "os"
    "testing"
)

func TestAppInfoSourcePath(t *testing.T) {
    tests := []struct {
        name     string
        appInfo  AppInfo
        expected string
    }{
        {
            name: "default path when empty",
            appInfo: AppInfo{
                Creator:               "user@example.com",
                Name:                  "my-app",
                DefaultSourceCodePath: "",
            },
            expected: "/Workspace/Users/user@example.com/my-app/",
        },
        {
            name: "custom path when set",
            appInfo: AppInfo{
                Creator:               "user@example.com",
                Name:                  "my-app",
                DefaultSourceCodePath: "/custom/path/to/app/",
            },
            expected: "/custom/path/to/app/",
        },
        {
            name: "handles special characters in creator",
            appInfo: AppInfo{
                Creator:               "user+test@example.com",
                Name:                  "test-app-123",
                DefaultSourceCodePath: "",
            },
            expected: "/Workspace/Users/user+test@example.com/test-app-123/",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.appInfo.SourcePath()
            if result != tt.expected {
                t.Errorf("expected %q, got %q", tt.expected, result)
            }
        })
    }
}

func TestResourcesSerialization(t *testing.T) {
    tests := []struct {
        name      string
        resources Resources
        wantJSON  string
    }{
        {
            name: "with warehouse",
            resources: Resources{
                Name:        "base",
                Description: "template resources",
                SQLWarehouse: &Warehouse{
                    ID:         "warehouse-123",
                    Permission: PermissionCanUse,
                },
            },
            wantJSON: `{"name":"base","description":"template resources","sql_warehouse":{"id":"warehouse-123","permission":"CAN_USE"}}`,
        },
        {
            name: "without warehouse",
            resources: Resources{
                Name:         "base",
                Description:  "no warehouse",
                SQLWarehouse: nil,
            },
            wantJSON: `{"name":"base","description":"no warehouse"}`,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test marshaling
            data, err := json.Marshal(tt.resources)
            if err != nil {
                t.Fatalf("failed to marshal: %v", err)
            }

            if string(data) != tt.wantJSON {
                t.Errorf("JSON mismatch:\nwant: %s\ngot:  %s", tt.wantJSON, string(data))
            }

            // Test unmarshaling
            var decoded Resources
            if err := json.Unmarshal(data, &decoded); err != nil {
                t.Fatalf("failed to unmarshal: %v", err)
            }

            if decoded.Name != tt.resources.Name {
                t.Errorf("name mismatch: want %q, got %q", tt.resources.Name, decoded.Name)
            }

            if tt.resources.SQLWarehouse != nil {
                if decoded.SQLWarehouse == nil {
                    t.Error("expected warehouse to be set")
                } else if decoded.SQLWarehouse.ID != tt.resources.SQLWarehouse.ID {
                    t.Errorf("warehouse ID mismatch: want %q, got %q",
                        tt.resources.SQLWarehouse.ID, decoded.SQLWarehouse.ID)
                }
            }
        })
    }
}

func TestResourcesFromEnv(t *testing.T) {
    tests := []struct {
        name        string
        envValue    string
        wantErr     bool
        wantID      string
    }{
        {
            name:     "valid warehouse ID",
            envValue: "warehouse-abc123",
            wantErr:  false,
            wantID:   "warehouse-abc123",
        },
        {
            name:     "missing warehouse ID",
            envValue: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set env var
            oldValue := os.Getenv("DATABRICKS_WAREHOUSE_ID")
            defer os.Setenv("DATABRICKS_WAREHOUSE_ID", oldValue)

            if tt.envValue != "" {
                os.Setenv("DATABRICKS_WAREHOUSE_ID", tt.envValue)
            } else {
                os.Unsetenv("DATABRICKS_WAREHOUSE_ID")
            }

            // Test function
            resources, err := ResourcesFromEnv()

            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if resources.SQLWarehouse == nil {
                t.Fatal("expected warehouse to be set")
            }

            if resources.SQLWarehouse.ID != tt.wantID {
                t.Errorf("warehouse ID mismatch: want %q, got %q", tt.wantID, resources.SQLWarehouse.ID)
            }

            if resources.SQLWarehouse.Permission != PermissionCanUse {
                t.Errorf("expected permission CAN_USE, got %q", resources.SQLWarehouse.Permission)
            }
        })
    }
}

func TestCreateAppMarshaling(t *testing.T) {
    app := CreateApp{
        Name:        "test-app",
        Description: "Test application",
        Resources: []Resources{
            {
                Name:        "base",
                Description: "resources",
                SQLWarehouse: &Warehouse{
                    ID:         "wh-123",
                    Permission: PermissionCanUse,
                },
            },
        },
    }

    data, err := json.Marshal(app)
    if err != nil {
        t.Fatalf("failed to marshal: %v", err)
    }

    var decoded CreateApp
    if err := json.Unmarshal(data, &decoded); err != nil {
        t.Fatalf("failed to unmarshal: %v", err)
    }

    if decoded.Name != app.Name {
        t.Errorf("name mismatch: want %q, got %q", app.Name, decoded.Name)
    }

    if len(decoded.Resources) != 1 {
        t.Fatalf("expected 1 resource, got %d", len(decoded.Resources))
    }
}

func TestUserInfoUnmarshaling(t *testing.T) {
    // Example JSON from Databricks CLI
    jsonData := `{
        "id": "12345",
        "active": true,
        "displayName": "John Doe",
        "userName": "john.doe@example.com"
    }`

    var userInfo UserInfo
    if err := json.Unmarshal([]byte(jsonData), &userInfo); err != nil {
        t.Fatalf("failed to unmarshal: %v", err)
    }

    if userInfo.ID != "12345" {
        t.Errorf("expected ID 12345, got %s", userInfo.ID)
    }

    if !userInfo.Active {
        t.Error("expected active to be true")
    }

    if userInfo.UserName != "john.doe@example.com" {
        t.Errorf("expected userName john.doe@example.com, got %s", userInfo.UserName)
    }
}
```

### Test File 2: Deployment Provider Tests

Create: `go-mcp/pkg/providers/deployment/provider_test.go`

```go
package deployment

import (
    "testing"
)

func TestFormatDeployResult(t *testing.T) {
    tests := []struct {
        name     string
        result   *DeployResult
        contains []string
    }{
        {
            name: "successful deployment",
            result: &DeployResult{
                Success: true,
                Message: "Deployment completed successfully",
                AppURL:  "https://example.com/app",
                AppName: "test-app",
            },
            contains: []string{
                "Successfully deployed app 'test-app'",
                "https://example.com/app",
                "Deployment completed successfully",
            },
        },
        {
            name: "failed deployment",
            result: &DeployResult{
                Success: false,
                Message: "Build failed",
                AppName: "test-app",
            },
            contains: []string{
                "Deployment failed for app 'test-app'",
                "Build failed",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            output := formatDeployResult(tt.result)

            for _, substr := range tt.contains {
                if !containsString(output, substr) {
                    t.Errorf("expected output to contain %q\ngot: %s", substr, output)
                }
            }
        })
    }
}

func containsString(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
        (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
         findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
```

### Test File 3: Integration Tests (Optional)

Create: `go-mcp/test/integration/deployment_test.go`

```go
// +build integration

package integration

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/appdotbuild/go-mcp/pkg/config"
    "github.com/appdotbuild/go-mcp/pkg/providers/databricks"
    "github.com/appdotbuild/go-mcp/pkg/providers/deployment"
    "github.com/appdotbuild/go-mcp/pkg/session"
    "log/slog"
)

func TestDeploymentWorkflow(t *testing.T) {
    // Skip if credentials not set
    if os.Getenv("DATABRICKS_HOST") == "" || os.Getenv("DATABRICKS_TOKEN") == "" {
        t.Skip("Skipping integration test: DATABRICKS_HOST and DATABRICKS_TOKEN not set")
    }

    if os.Getenv("DATABRICKS_WAREHOUSE_ID") == "" {
        t.Skip("Skipping integration test: DATABRICKS_WAREHOUSE_ID not set")
    }

    // Create temporary test directory
    tmpDir := t.TempDir()

    // Setup would include:
    // 1. Create test app structure
    // 2. Scaffold app
    // 3. Validate app
    // 4. Deploy app
    // 5. Verify deployment
    // 6. Cleanup

    t.Log("Integration test placeholder - implement with real Databricks environment")
}

func TestGetUserInfo(t *testing.T) {
    if os.Getenv("DATABRICKS_HOST") == "" || os.Getenv("DATABRICKS_TOKEN") == "" {
        t.Skip("Skipping integration test: credentials not set")
    }

    userInfo, err := databricks.GetUserInfo()
    if err != nil {
        t.Fatalf("failed to get user info: %v", err)
    }

    if userInfo.UserName == "" {
        t.Error("expected non-empty username")
    }

    t.Logf("Current user: %s (%s)", userInfo.DisplayName, userInfo.UserName)
}
```

## Running Tests

### Unit Tests
```bash
cd go-mcp

# Run all unit tests
go test ./pkg/providers/databricks/...
go test ./pkg/providers/deployment/...

# Run with coverage
go test -coverprofile=coverage.out ./pkg/providers/databricks/...
go test -coverprofile=coverage.out ./pkg/providers/deployment/...

# View coverage
go tool cover -html=coverage.out
```

### Integration Tests
```bash
# Set credentials
export DATABRICKS_HOST=https://your-workspace.databricks.com
export DATABRICKS_TOKEN=dapi...
export DATABRICKS_WAREHOUSE_ID=your-warehouse-id

# Run integration tests
go test -tags=integration ./test/integration/...
```

### Using Make
```bash
# Run all tests
make test

# Run with race detector
make test-race

# Run with coverage
make test-coverage
```

## Success Criteria
- [ ] All unit tests pass
- [ ] Code coverage > 80% for core logic
- [ ] Tests follow Go table-driven pattern
- [ ] Integration tests run successfully (with credentials)
- [ ] No test failures
- [ ] `make test` passes
- [ ] `make test-race` passes without race conditions

## Coverage Goals

Aim for these coverage levels:
- `pkg/providers/databricks/deployment.go`: > 80%
- `pkg/providers/deployment/provider.go`: > 75%
- Overall package coverage: > 70%
