package vscode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultSettingsPath_VSCode_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	ctx := context.Background()
	ctx = env.Set(ctx, "HOME", "/home/testuser")

	path, err := getDefaultSettingsPath(ctx, vscodeIDE)
	require.NoError(t, err)
	assert.Equal(t, "/home/testuser/.config/Code/User/settings.json", path)
}

func TestGetDefaultSettingsPath_Cursor_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	ctx := context.Background()
	ctx = env.Set(ctx, "HOME", "/home/testuser")

	path, err := getDefaultSettingsPath(ctx, cursorIDE)
	require.NoError(t, err)
	assert.Equal(t, "/home/testuser/.config/Cursor/User/settings.json", path)
}

func TestGetDefaultSettingsPath_VSCode_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	ctx := context.Background()
	ctx = env.Set(ctx, "APPDATA", `C:\Users\testuser\AppData\Roaming`)

	path, err := getDefaultSettingsPath(ctx, vscodeIDE)
	require.NoError(t, err)
	assert.Equal(t, `C:\Users\testuser\AppData\Roaming\Code\User\settings.json`, path)
}

func TestGetDefaultSettingsPath_Cursor_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	ctx := context.Background()
	ctx = env.Set(ctx, "APPDATA", `C:\Users\testuser\AppData\Roaming`)

	path, err := getDefaultSettingsPath(ctx, cursorIDE)
	require.NoError(t, err)
	assert.Equal(t, `C:\Users\testuser\AppData\Roaming\Cursor\User\settings.json`, path)
}

func TestLoadSettings_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	settingsData := map[string]any{
		"editor.fontSize": 14,
		"remote.SSH.serverPickPortsFromRange": map[string]any{
			"test-conn": "4000-4005",
		},
	}
	data, err := json.Marshal(settingsData)
	require.NoError(t, err)
	err = os.WriteFile(settingsPath, data, 0o600)
	require.NoError(t, err)

	settings, err := loadSettings(settingsPath)
	require.NoError(t, err)
	assert.InDelta(t, float64(14), settings["editor.fontSize"], 0.01)
	assert.Contains(t, settings, "remote.SSH.serverPickPortsFromRange")
}

func TestLoadSettings_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	err := os.WriteFile(settingsPath, []byte("invalid json {"), 0o600)
	require.NoError(t, err)

	_, err = loadSettings(settingsPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse settings JSON")
}

func TestLoadSettings_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// JSONC format with comments and trailing commas (typical VS Code settings)
	settingsData := `{
		// Editor settings
		"editor.fontSize": 14,
		/* Connection settings */
		"remote.SSH.serverPickPortsFromRange": {
			"test-conn": "4000-4005" // Port range for SSH
		},
		"remote.SSH.remotePlatform": {
			"test-conn": "linux", // trailing comma
		}
	}`
	err := os.WriteFile(settingsPath, []byte(settingsData), 0o600)
	require.NoError(t, err)

	settings, err := loadSettings(settingsPath)
	require.NoError(t, err)
	assert.InDelta(t, float64(14), settings["editor.fontSize"], 0.01)
	assert.Contains(t, settings, "remote.SSH.serverPickPortsFromRange")

	portRangeObj := settings["remote.SSH.serverPickPortsFromRange"].(map[string]any)
	assert.Equal(t, "4000-4005", portRangeObj["test-conn"])
}

func TestLoadSettings_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "nonexistent.json")

	_, err := loadSettings(settingsPath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestValidateSettings_Complete(t *testing.T) {
	settings := map[string]any{
		"remote.SSH.serverPickPortsFromRange": map[string]any{
			"test-conn": "4000-4005",
		},
		"remote.SSH.remotePlatform": map[string]any{
			"test-conn": "linux",
		},
		"remote.SSH.remoteServerListenOnSocket": true,
		"remote.SSH.defaultExtensions": []any{
			"ms-python.python",
			"ms-toolsai.jupyter",
		},
	}

	missing := validateSettings(settings, "test-conn")
	assert.True(t, missing.isEmpty())
}

func TestValidateSettings_Missing(t *testing.T) {
	settings := map[string]any{}

	missing := validateSettings(settings, "test-conn")
	assert.False(t, missing.isEmpty())
	assert.True(t, missing.portRange)
	assert.True(t, missing.platform)
	assert.Equal(t, []string{"ms-python.python", "ms-toolsai.jupyter"}, missing.extensions)
}

func TestValidateSettings_IncorrectValues(t *testing.T) {
	settings := map[string]any{
		"remote.SSH.serverPickPortsFromRange": map[string]any{
			"test-conn": "5000-5005", // Wrong port range
		},
		"remote.SSH.remotePlatform": map[string]any{
			"test-conn": "windows", // Wrong platform
		},
		"remote.SSH.defaultExtensions": []any{
			"ms-python.python", // Missing jupyter
		},
	}

	missing := validateSettings(settings, "test-conn")
	assert.False(t, missing.isEmpty())
	assert.True(t, missing.portRange)
	assert.True(t, missing.platform)
	assert.Equal(t, []string{"ms-toolsai.jupyter"}, missing.extensions)
}

func TestValidateSettings_MissingConnection(t *testing.T) {
	settings := map[string]any{
		"remote.SSH.serverPickPortsFromRange": map[string]any{
			"other-conn": "4000-4005",
		},
		"remote.SSH.remotePlatform": map[string]any{
			"other-conn": "linux",
		},
		"remote.SSH.defaultExtensions": []any{
			"ms-python.python",
			"ms-toolsai.jupyter",
		},
	}

	// Validating for a different connection should show port and platform as missing
	missing := validateSettings(settings, "test-conn")
	assert.False(t, missing.isEmpty())
	assert.True(t, missing.portRange)
	assert.True(t, missing.platform)
	assert.Empty(t, missing.extensions) // Extensions are global, so they're present
}

func TestUpdateSettings_PreserveExistingConnections(t *testing.T) {
	settings := map[string]any{
		"remote.SSH.serverPickPortsFromRange": map[string]any{
			"conn-a": "5000-5005",
			"conn-b": "6000-6005",
		},
		"remote.SSH.remotePlatform": map[string]any{
			"conn-a": "linux",
			"conn-b": "darwin",
		},
		"remote.SSH.defaultExtensions": []any{
			"other.extension",
		},
	}

	missing := &missingSettings{
		portRange:  true,
		platform:   true,
		extensions: []string{"ms-python.python", "ms-toolsai.jupyter"},
	}

	updateSettings(settings, "conn-c", missing)

	// Check that new connection was added
	portRangeObj := settings["remote.SSH.serverPickPortsFromRange"].(map[string]any)
	assert.Equal(t, "4000-4005", portRangeObj["conn-c"])

	platformObj := settings["remote.SSH.remotePlatform"].(map[string]any)
	assert.Equal(t, "linux", platformObj["conn-c"])

	// Check that existing connections were preserved
	assert.Equal(t, "5000-5005", portRangeObj["conn-a"])
	assert.Equal(t, "6000-6005", portRangeObj["conn-b"])
	assert.Equal(t, "linux", platformObj["conn-a"])
	assert.Equal(t, "darwin", platformObj["conn-b"])

	// Check that extensions were merged
	extArray := settings["remote.SSH.defaultExtensions"].([]any)
	assert.Len(t, extArray, 3)
	assert.Contains(t, extArray, "other.extension")
	assert.Contains(t, extArray, "ms-python.python")
	assert.Contains(t, extArray, "ms-toolsai.jupyter")
}

func TestUpdateSettings_NewConnection(t *testing.T) {
	settings := map[string]any{}

	missing := &missingSettings{
		portRange:  true,
		platform:   true,
		extensions: []string{"ms-python.python", "ms-toolsai.jupyter"},
	}

	updateSettings(settings, "new-conn", missing)

	portRangeObj := settings["remote.SSH.serverPickPortsFromRange"].(map[string]any)
	assert.Equal(t, "4000-4005", portRangeObj["new-conn"])

	platformObj := settings["remote.SSH.remotePlatform"].(map[string]any)
	assert.Equal(t, "linux", platformObj["new-conn"])

	extArray := settings["remote.SSH.defaultExtensions"].([]any)
	assert.Len(t, extArray, 2)
	assert.Contains(t, extArray, "ms-python.python")
	assert.Contains(t, extArray, "ms-toolsai.jupyter")
}

func TestUpdateSettings_GlobalExtensions(t *testing.T) {
	// Verify that extensions are global, not per-connection
	settings := map[string]any{
		"remote.SSH.defaultExtensions": []any{
			"ms-python.python",
		},
	}

	missing := &missingSettings{
		extensions: []string{"ms-toolsai.jupyter"},
	}

	updateSettings(settings, "conn-a", missing)

	extArray := settings["remote.SSH.defaultExtensions"].([]any)
	assert.Len(t, extArray, 2)
	assert.Contains(t, extArray, "ms-python.python")
	assert.Contains(t, extArray, "ms-toolsai.jupyter")

	// Update for another connection should use the same global array
	missing2 := &missingSettings{
		extensions: []string{"another.extension"},
	}

	updateSettings(settings, "conn-b", missing2)

	extArray = settings["remote.SSH.defaultExtensions"].([]any)
	assert.Len(t, extArray, 3)
	assert.Contains(t, extArray, "ms-python.python")
	assert.Contains(t, extArray, "ms-toolsai.jupyter")
	assert.Contains(t, extArray, "another.extension")
}

func TestUpdateSettings_MergeExtensions(t *testing.T) {
	settings := map[string]any{
		"remote.SSH.defaultExtensions": []any{
			"existing.extension",
			"ms-python.python",
		},
	}

	missing := &missingSettings{
		extensions: []string{"ms-python.python", "ms-toolsai.jupyter"},
	}

	updateSettings(settings, "test-conn", missing)

	extArray := settings["remote.SSH.defaultExtensions"].([]any)
	assert.Len(t, extArray, 3)
	assert.Contains(t, extArray, "existing.extension")
	assert.Contains(t, extArray, "ms-python.python")
	assert.Contains(t, extArray, "ms-toolsai.jupyter")
}

func TestUpdateSettings_PartialUpdate(t *testing.T) {
	settings := map[string]any{
		"remote.SSH.serverPickPortsFromRange": map[string]any{
			"test-conn": "4000-4005", // Already correct
		},
		"remote.SSH.remotePlatform": map[string]any{
			"other-conn": "linux",
		},
		"remote.SSH.defaultExtensions": []any{
			"ms-python.python",
			"ms-toolsai.jupyter",
		},
	}

	missing := &missingSettings{
		portRange:  false, // Already set
		platform:   true,  // Needs update
		extensions: nil,   // Already present
	}

	updateSettings(settings, "test-conn", missing)

	// Port range should not be modified
	portRangeObj := settings["remote.SSH.serverPickPortsFromRange"].(map[string]any)
	assert.Equal(t, "4000-4005", portRangeObj["test-conn"])

	// Platform should be added for test-conn
	platformObj := settings["remote.SSH.remotePlatform"].(map[string]any)
	assert.Equal(t, "linux", platformObj["test-conn"])
	assert.Equal(t, "linux", platformObj["other-conn"]) // Preserve other connection

	// Extensions should not be modified
	extArray := settings["remote.SSH.defaultExtensions"].([]any)
	assert.Len(t, extArray, 2)
}

func TestBackupSettings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	originalContent := []byte(`{"key": "value"}`)
	err := os.WriteFile(settingsPath, originalContent, 0o600)
	require.NoError(t, err)

	ctx, _ := cmdio.NewTestContextWithStderr(context.Background())
	err = backupSettings(ctx, settingsPath)
	require.NoError(t, err)

	backupPath := settingsPath + ".bak"
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, backupContent)
}

func TestSaveSettings_Formatting(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	settings := map[string]any{
		"remote.SSH.serverPickPortsFromRange": map[string]any{
			"test-conn": "4000-4005",
		},
		"editor.fontSize": 14,
	}

	err := saveSettings(settingsPath, settings)
	require.NoError(t, err)

	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed map[string]any
	err = json.Unmarshal(content, &parsed)
	require.NoError(t, err)

	// Verify formatting (should have 2-space indent)
	assert.Contains(t, string(content), "  \"remote.SSH.serverPickPortsFromRange\"")

	// Verify permissions
	info, err := os.Stat(settingsPath)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestMissingSettings_IsEmpty(t *testing.T) {
	empty := &missingSettings{}
	assert.True(t, empty.isEmpty())

	notEmpty := &missingSettings{portRange: true}
	assert.False(t, notEmpty.isEmpty())

	notEmpty2 := &missingSettings{extensions: []string{"ext"}}
	assert.False(t, notEmpty2.isEmpty())
}

func TestGetManualInstructions_VSCode(t *testing.T) {
	instructions := GetManualInstructions(vscodeIDE, "test-conn")

	assert.Contains(t, instructions, "VS Code")
	assert.Contains(t, instructions, "test-conn")
	assert.Contains(t, instructions, "4000-4005")
	assert.Contains(t, instructions, "linux")
	assert.Contains(t, instructions, "ms-python.python")
	assert.Contains(t, instructions, "ms-toolsai.jupyter")
	assert.Contains(t, instructions, "remote.SSH.serverPickPortsFromRange")
	assert.Contains(t, instructions, "remote.SSH.remotePlatform")
	assert.Contains(t, instructions, "remote.SSH.defaultExtensions")
}

func TestGetManualInstructions_Cursor(t *testing.T) {
	instructions := GetManualInstructions("cursor", "my-connection")

	assert.Contains(t, instructions, "Cursor")
	assert.Contains(t, instructions, "my-connection")
	assert.Contains(t, instructions, "4000-4005")
	assert.Contains(t, instructions, "linux")
	assert.Contains(t, instructions, "ms-python.python")
	assert.Contains(t, instructions, "ms-toolsai.jupyter")
}
