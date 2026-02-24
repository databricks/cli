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
	"github.com/tailscale/hujson"
)

func parseTestValue(t *testing.T, jsonStr string) hujson.Value {
	t.Helper()
	v, err := hujson.Parse([]byte(jsonStr))
	require.NoError(t, err)
	return v
}

func findString(t *testing.T, v hujson.Value, ptr string) (string, bool) {
	t.Helper()
	found := v.Find(ptr)
	if found == nil {
		return "", false
	}
	var s string
	if err := json.Unmarshal(found.Pack(), &s); err != nil {
		return "", false
	}
	return s, true
}

func findStringSlice(t *testing.T, v hujson.Value, ptr string) []string {
	t.Helper()
	found := v.Find(ptr)
	if found == nil {
		return nil
	}
	var ss []string
	if err := json.Unmarshal(found.Pack(), &ss); err != nil {
		return nil
	}
	return ss
}

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

func TestGetDefaultSettingsPath_VSCode_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Darwin-specific test")
	}

	ctx := context.Background()
	ctx = env.Set(ctx, "HOME", "/Users/testuser")

	path, err := getDefaultSettingsPath(ctx, vscodeIDE)
	require.NoError(t, err)
	assert.Equal(t, "/Users/testuser/Library/Application Support/Code/User/settings.json", path)
}

func TestGetDefaultSettingsPath_Cursor_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Darwin-specific test")
	}

	ctx := context.Background()
	ctx = env.Set(ctx, "HOME", "/Users/testuser")

	path, err := getDefaultSettingsPath(ctx, cursorIDE)
	require.NoError(t, err)
	assert.Equal(t, "/Users/testuser/Library/Application Support/Cursor/User/settings.json", path)
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

	settingsData := `{
		"editor.fontSize": 14,
		"remote.SSH.serverPickPortsFromRange": {
			"test-conn": "4000-4005"
		}
	}`
	err := os.WriteFile(settingsPath, []byte(settingsData), 0o600)
	require.NoError(t, err)

	settings, err := loadSettings(settingsPath)
	require.NoError(t, err)
	assert.NotNil(t, settings.Find("/editor.fontSize"))
	assert.NotNil(t, settings.Find("/remote.SSH.serverPickPortsFromRange"))
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
	assert.NotNil(t, settings.Find("/editor.fontSize"))
	assert.NotNil(t, settings.Find("/remote.SSH.serverPickPortsFromRange"))

	val, ok := findString(t, settings, jsonPtr(serverPickPortsKey, "test-conn"))
	assert.True(t, ok)
	assert.Equal(t, "4000-4005", val)
}

func TestLoadSettings_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "nonexistent.json")

	_, err := loadSettings(settingsPath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestValidateSettings_Complete(t *testing.T) {
	v := parseTestValue(t, `{
		"remote.SSH.serverPickPortsFromRange": {"test-conn": "4000-4005"},
		"remote.SSH.remotePlatform": {"test-conn": "linux"},
		"remote.SSH.remoteServerListenOnSocket": true,
		"remote.SSH.defaultExtensions": ["ms-python.python", "ms-toolsai.jupyter"]
	}`)

	missing := validateSettings(v, "test-conn")
	assert.True(t, missing.isEmpty())
}

func TestValidateSettings_Missing(t *testing.T) {
	v := parseTestValue(t, `{}`)

	missing := validateSettings(v, "test-conn")
	assert.False(t, missing.isEmpty())
	assert.True(t, missing.portRange)
	assert.True(t, missing.platform)
	assert.Equal(t, []string{"ms-python.python", "ms-toolsai.jupyter"}, missing.extensions)
}

func TestValidateSettings_IncorrectValues(t *testing.T) {
	v := parseTestValue(t, `{
		"remote.SSH.serverPickPortsFromRange": {"test-conn": "5000-5005"},
		"remote.SSH.remotePlatform": {"test-conn": "windows"},
		"remote.SSH.defaultExtensions": ["ms-python.python"]
	}`)

	missing := validateSettings(v, "test-conn")
	assert.False(t, missing.isEmpty())
	assert.True(t, missing.portRange)
	assert.True(t, missing.platform)
	assert.Equal(t, []string{"ms-toolsai.jupyter"}, missing.extensions)
}

func TestValidateSettings_DuplicateExtensionsNotReported(t *testing.T) {
	v := parseTestValue(t, `{
		"remote.SSH.serverPickPortsFromRange": {"test-conn": "4000-4005"},
		"remote.SSH.remotePlatform": {"test-conn": "linux"},
		"remote.SSH.remoteServerListenOnSocket": true,
		"remote.SSH.defaultExtensions": ["ms-python.python", "ms-python.python", "ms-toolsai.jupyter"]
	}`)

	missing := validateSettings(v, "test-conn")
	assert.True(t, missing.isEmpty())
}

func TestValidateSettings_MissingConnection(t *testing.T) {
	v := parseTestValue(t, `{
		"remote.SSH.serverPickPortsFromRange": {"other-conn": "4000-4005"},
		"remote.SSH.remotePlatform": {"other-conn": "linux"},
		"remote.SSH.defaultExtensions": ["ms-python.python", "ms-toolsai.jupyter"]
	}`)

	// Validating for a different connection should show port and platform as missing
	missing := validateSettings(v, "test-conn")
	assert.False(t, missing.isEmpty())
	assert.True(t, missing.portRange)
	assert.True(t, missing.platform)
	assert.Empty(t, missing.extensions) // Extensions are global, so they're present
}

func TestUpdateSettings_PreserveExistingConnections(t *testing.T) {
	v := parseTestValue(t, `{
		"remote.SSH.serverPickPortsFromRange": {
			"conn-a": "5000-5005",
			"conn-b": "6000-6005"
		},
		"remote.SSH.remotePlatform": {
			"conn-a": "linux",
			"conn-b": "darwin"
		},
		"remote.SSH.defaultExtensions": ["other.extension"]
	}`)

	missing := &missingSettings{
		portRange:  true,
		platform:   true,
		extensions: []string{"ms-python.python", "ms-toolsai.jupyter"},
	}

	err := updateSettings(&v, "conn-c", missing)
	require.NoError(t, err)

	// Check that new connection was added
	val, ok := findString(t, v, jsonPtr(serverPickPortsKey, "conn-c"))
	assert.True(t, ok)
	assert.Equal(t, "4000-4005", val)

	val, ok = findString(t, v, jsonPtr(remotePlatformKey, "conn-c"))
	assert.True(t, ok)
	assert.Equal(t, "linux", val)

	// Check that existing connections were preserved
	val, ok = findString(t, v, jsonPtr(serverPickPortsKey, "conn-a"))
	assert.True(t, ok)
	assert.Equal(t, "5000-5005", val)

	val, ok = findString(t, v, jsonPtr(serverPickPortsKey, "conn-b"))
	assert.True(t, ok)
	assert.Equal(t, "6000-6005", val)

	val, ok = findString(t, v, jsonPtr(remotePlatformKey, "conn-a"))
	assert.True(t, ok)
	assert.Equal(t, "linux", val)

	val, ok = findString(t, v, jsonPtr(remotePlatformKey, "conn-b"))
	assert.True(t, ok)
	assert.Equal(t, "darwin", val)

	// Check that extensions were merged
	exts := findStringSlice(t, v, jsonPtr(defaultExtensionsKey))
	assert.Len(t, exts, 3)
	assert.Contains(t, exts, "other.extension")
	assert.Contains(t, exts, "ms-python.python")
	assert.Contains(t, exts, "ms-toolsai.jupyter")
}

func TestUpdateSettings_NewConnection(t *testing.T) {
	v := parseTestValue(t, `{}`)

	missing := &missingSettings{
		portRange:  true,
		platform:   true,
		extensions: []string{"ms-python.python", "ms-toolsai.jupyter"},
	}

	err := updateSettings(&v, "new-conn", missing)
	require.NoError(t, err)

	val, ok := findString(t, v, jsonPtr(serverPickPortsKey, "new-conn"))
	assert.True(t, ok)
	assert.Equal(t, "4000-4005", val)

	val, ok = findString(t, v, jsonPtr(remotePlatformKey, "new-conn"))
	assert.True(t, ok)
	assert.Equal(t, "linux", val)

	exts := findStringSlice(t, v, jsonPtr(defaultExtensionsKey))
	assert.Len(t, exts, 2)
	assert.Contains(t, exts, "ms-python.python")
	assert.Contains(t, exts, "ms-toolsai.jupyter")
}

func TestUpdateSettings_GlobalExtensions(t *testing.T) {
	// Verify that extensions are global, not per-connection
	v := parseTestValue(t, `{
		"remote.SSH.defaultExtensions": ["ms-python.python"]
	}`)

	missing := &missingSettings{
		extensions: []string{"ms-toolsai.jupyter"},
	}

	err := updateSettings(&v, "conn-a", missing)
	require.NoError(t, err)

	exts := findStringSlice(t, v, jsonPtr(defaultExtensionsKey))
	assert.Len(t, exts, 2)
	assert.Contains(t, exts, "ms-python.python")
	assert.Contains(t, exts, "ms-toolsai.jupyter")

	// Update for another connection should use the same global array
	missing2 := &missingSettings{
		extensions: []string{"another.extension"},
	}

	err = updateSettings(&v, "conn-b", missing2)
	require.NoError(t, err)

	exts = findStringSlice(t, v, jsonPtr(defaultExtensionsKey))
	assert.Len(t, exts, 3)
	assert.Contains(t, exts, "ms-python.python")
	assert.Contains(t, exts, "ms-toolsai.jupyter")
	assert.Contains(t, exts, "another.extension")
}

func TestUpdateSettings_MergeExtensions(t *testing.T) {
	v := parseTestValue(t, `{
		"remote.SSH.defaultExtensions": ["existing.extension", "ms-python.python"]
	}`)

	missing := &missingSettings{
		extensions: []string{"ms-toolsai.jupyter"}, // ms-python.python already present
	}

	err := updateSettings(&v, "test-conn", missing)
	require.NoError(t, err)

	exts := findStringSlice(t, v, jsonPtr(defaultExtensionsKey))
	assert.Len(t, exts, 3)
	assert.Contains(t, exts, "existing.extension")
	assert.Contains(t, exts, "ms-python.python")
	assert.Contains(t, exts, "ms-toolsai.jupyter")
}

func TestUpdateSettings_PartialUpdate(t *testing.T) {
	v := parseTestValue(t, `{
		"remote.SSH.serverPickPortsFromRange": {"test-conn": "4000-4005"},
		"remote.SSH.remotePlatform": {"other-conn": "linux"},
		"remote.SSH.defaultExtensions": ["ms-python.python", "ms-toolsai.jupyter"]
	}`)

	missing := &missingSettings{
		portRange:  false, // Already set
		platform:   true,  // Needs update
		extensions: nil,   // Already present
	}

	err := updateSettings(&v, "test-conn", missing)
	require.NoError(t, err)

	// Port range should not be modified
	val, ok := findString(t, v, jsonPtr(serverPickPortsKey, "test-conn"))
	assert.True(t, ok)
	assert.Equal(t, "4000-4005", val)

	// Platform should be added for test-conn
	val, ok = findString(t, v, jsonPtr(remotePlatformKey, "test-conn"))
	assert.True(t, ok)
	assert.Equal(t, "linux", val)

	val, ok = findString(t, v, jsonPtr(remotePlatformKey, "other-conn"))
	assert.True(t, ok)
	assert.Equal(t, "linux", val) // Preserve other connection

	// Extensions should not be modified
	exts := findStringSlice(t, v, jsonPtr(defaultExtensionsKey))
	assert.Len(t, exts, 2)
}

func TestBackupSettings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	originalBak := settingsPath + ".original.bak"
	latestBak := settingsPath + ".latest.bak"

	originalContent := []byte(`{"key": "value"}`)
	err := os.WriteFile(settingsPath, originalContent, 0o600)
	require.NoError(t, err)

	ctx, _ := cmdio.NewTestContextWithStderr(context.Background())

	// First backup: should create .original.bak
	err = backupSettings(ctx, settingsPath)
	require.NoError(t, err)

	content, err := os.ReadFile(originalBak)
	require.NoError(t, err)
	assert.Equal(t, originalContent, content)
	_, err = os.Stat(latestBak)
	assert.True(t, os.IsNotExist(err))

	// Second backup: .original.bak exists, should create .latest.bak
	updatedContent := []byte(`{"key": "updated"}`)
	err = os.WriteFile(settingsPath, updatedContent, 0o600)
	require.NoError(t, err)

	err = backupSettings(ctx, settingsPath)
	require.NoError(t, err)

	// .original.bak must remain unchanged
	content, err = os.ReadFile(originalBak)
	require.NoError(t, err)
	assert.Equal(t, originalContent, content)

	// .latest.bak should have the updated content
	content, err = os.ReadFile(latestBak)
	require.NoError(t, err)
	assert.Equal(t, updatedContent, content)
}

func TestSaveSettings_Formatting(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	v := parseTestValue(t, `{
		"remote.SSH.serverPickPortsFromRange": {"test-conn": "4000-4005"},
		"editor.fontSize": 14
	}`)

	err := saveSettings(settingsPath, &v)
	require.NoError(t, err)

	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	// Verify it's valid JSON after standardizing
	standardized, err := hujson.Standardize(content)
	require.NoError(t, err)
	var parsed map[string]any
	err = json.Unmarshal(standardized, &parsed)
	require.NoError(t, err)

	// Verify permissions
	info, err := os.Stat(settingsPath)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestSaveSettings_PreservesComments(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	original := `{
	// This is a comment
	"editor.fontSize": 14
}`
	err := os.WriteFile(settingsPath, []byte(original), 0o600)
	require.NoError(t, err)

	v, err := loadSettings(settingsPath)
	require.NoError(t, err)

	// Add a new setting
	missing := &missingSettings{listenOnSocket: true}
	err = updateSettings(&v, "test-conn", missing)
	require.NoError(t, err)

	err = saveSettings(settingsPath, &v)
	require.NoError(t, err)

	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "// This is a comment")
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
