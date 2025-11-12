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
			data, err := json.Marshal(tt.resources)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			if string(data) != tt.wantJSON {
				t.Errorf("JSON mismatch:\nwant: %s\ngot:  %s", tt.wantJSON, string(data))
			}

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
		name     string
		envValue string
		wantErr  bool
		wantID   string
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
			oldValue := os.Getenv("DATABRICKS_WAREHOUSE_ID")
			defer os.Setenv("DATABRICKS_WAREHOUSE_ID", oldValue)

			if tt.envValue != "" {
				os.Setenv("DATABRICKS_WAREHOUSE_ID", tt.envValue)
			} else {
				os.Unsetenv("DATABRICKS_WAREHOUSE_ID")
			}

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

func TestCreateAppRequestMarshaling(t *testing.T) {
	app := CreateAppRequest{
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

	var decoded CreateAppRequest
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

func TestPermissionConstants(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		expected   string
	}{
		{
			name:       "can use permission",
			permission: PermissionCanUse,
			expected:   "CAN_USE",
		},
		{
			name:       "can manage permission",
			permission: PermissionCanManage,
			expected:   "CAN_MANAGE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.permission) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.permission))
			}
		})
	}
}
