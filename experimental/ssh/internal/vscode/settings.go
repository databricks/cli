package vscode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/tidwall/jsonc"
)

const (
	portRange            = "4000-4005"
	remotePlatform       = "linux"
	pythonExtension      = "ms-python.python"
	jupyterExtension     = "ms-toolsai.jupyter"
	serverPickPortsKey   = "remote.SSH.serverPickPortsFromRange"
	remotePlatformKey    = "remote.SSH.remotePlatform"
	defaultExtensionsKey = "remote.SSH.defaultExtensions"
	listenOnSocketKey    = "remote.SSH.remoteServerListenOnSocket"
	vscodeIDE            = "vscode"
	cursorIDE            = "cursor"
	vscodeName           = "VS Code"
	cursorName           = "Cursor"
)

func getIDEName(ide string) string {
	if ide == cursorIDE {
		return cursorName
	}
	return vscodeName
}

type missingSettings struct {
	portRange      bool
	platform       bool
	listenOnSocket bool
	extensions     []string
}

func (m *missingSettings) isEmpty() bool {
	return !m.portRange && !m.platform && !m.listenOnSocket && len(m.extensions) == 0
}

func CheckAndUpdateSettings(ctx context.Context, ide, connectionName string) error {
	if !cmdio.IsPromptSupported(ctx) {
		log.Debugf(ctx, "Skipping IDE settings check: prompts not supported")
		return nil
	}

	settingsPath, err := getDefaultSettingsPath(ctx, ide)
	if err != nil {
		return fmt.Errorf("failed to get settings path: %w", err)
	}

	settings, err := loadSettings(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return handleMissingFile(ctx, ide, connectionName, settingsPath)
		}
		return fmt.Errorf("failed to load settings: %w", err)
	}

	missing := validateSettings(settings, connectionName)
	if missing.isEmpty() {
		log.Debugf(ctx, "IDE settings already correct for %s", connectionName)
		return nil
	}

	shouldUpdate, err := promptUserForUpdate(ctx, ide, connectionName, missing)
	if err != nil {
		return fmt.Errorf("failed to prompt user: %w", err)
	}
	if !shouldUpdate {
		log.Infof(ctx, "Skipping IDE settings update")
		return nil
	}

	if err := backupSettings(ctx, settingsPath); err != nil {
		log.Warnf(ctx, "Failed to backup settings: %v. Continuing with update.", err)
	}

	updateSettings(settings, connectionName, missing)

	if err := saveSettings(settingsPath, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Updated %s settings for '%s'", getIDEName(ide), connectionName))
	return nil
}

func getDefaultSettingsPath(ctx context.Context, ide string) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	appName := "Code"
	if ide == cursorIDE {
		appName = "Cursor"
	}

	var settingsDir string
	switch runtime.GOOS {
	case "darwin":
		settingsDir = filepath.Join(home, "Library", "Application Support", appName, "User")
	case "windows":
		appData := env.Get(ctx, "APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		settingsDir = filepath.Join(appData, appName, "User")
	case "linux":
		settingsDir = filepath.Join(home, ".config", appName, "User")
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return filepath.Join(settingsDir, "settings.json"), nil
}

func loadSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// VS Code/Cursor settings files are in JSONC format (JSON with comments).
	cleanJSON := jsonc.ToJSON(data)
	var settings map[string]any
	if err := json.Unmarshal(cleanJSON, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}
	return settings, nil
}

func hasCorrectPortRange(settings map[string]any, connectionName string) bool {
	portRangeObj, ok := settings[serverPickPortsKey].(map[string]any)
	if !ok {
		return false
	}
	val, ok := portRangeObj[connectionName].(string)
	return ok && val == portRange
}

func hasCorrectPlatform(settings map[string]any, connectionName string) bool {
	platformObj, ok := settings[remotePlatformKey].(map[string]any)
	if !ok {
		return false
	}
	val, ok := platformObj[connectionName].(string)
	return ok && val == remotePlatform
}

func hasCorrectListenOnSocket(settings map[string]any) bool {
	val, ok := settings[listenOnSocketKey].(bool)
	return ok && val
}

func getMissingExtensions(settings map[string]any) []string {
	requiredExtensions := []string{pythonExtension, jupyterExtension}

	extArray, ok := settings[defaultExtensionsKey].([]any)
	if !ok {
		return requiredExtensions
	}

	existingExts := make(map[string]bool)
	for _, ext := range extArray {
		if extStr, ok := ext.(string); ok {
			existingExts[extStr] = true
		}
	}

	var missing []string
	for _, reqExt := range requiredExtensions {
		if !existingExts[reqExt] {
			missing = append(missing, reqExt)
		}
	}
	return missing
}

func validateSettings(settings map[string]any, connectionName string) *missingSettings {
	return &missingSettings{
		portRange:      !hasCorrectPortRange(settings, connectionName),
		platform:       !hasCorrectPlatform(settings, connectionName),
		listenOnSocket: !hasCorrectListenOnSocket(settings),
		extensions:     getMissingExtensions(settings),
	}
}

func promptUserForUpdate(ctx context.Context, ide, connectionName string, _ *missingSettings) (bool, error) {
	question := fmt.Sprintf("%s settings are missing required configuration for '%s'. Update settings?", getIDEName(ide), connectionName)
	return cmdio.AskYesOrNo(ctx, question)
}

func handleMissingFile(ctx context.Context, ide, connectionName, settingsPath string) error {
	question := fmt.Sprintf("%s settings not found. Create settings with recommended configuration for '%s'?", getIDEName(ide), connectionName)
	shouldCreate, err := cmdio.AskYesOrNo(ctx, question)
	if err != nil {
		return fmt.Errorf("failed to prompt user: %w", err)
	}
	if !shouldCreate {
		log.Infof(ctx, "Skipping IDE settings creation")
		return nil
	}

	settingsDir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	settings := make(map[string]any)
	missing := &missingSettings{
		portRange:      true,
		platform:       true,
		listenOnSocket: true,
		extensions:     []string{pythonExtension, jupyterExtension},
	}
	updateSettings(settings, connectionName, missing)

	if err := saveSettings(settingsPath, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Created %s settings at %s", getIDEName(ide), filepath.ToSlash(settingsPath)))
	return nil
}

func backupSettings(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	backupPath := path + ".bak"
	log.Infof(ctx, "Backing up settings to %s", filepath.ToSlash(backupPath))
	return os.WriteFile(backupPath, data, 0o600)
}

func getOrDefault[T any](settings map[string]any, key string, defaultVal T) T {
	if existing, ok := settings[key].(T); ok {
		return existing
	}
	return defaultVal
}

func updateSettings(settings map[string]any, connectionName string, missing *missingSettings) {
	if missing.portRange {
		portsConfig := getOrDefault(settings, serverPickPortsKey, make(map[string]any))
		portsConfig[connectionName] = portRange
		settings[serverPickPortsKey] = portsConfig
	}

	if missing.platform {
		platformConfig := getOrDefault(settings, remotePlatformKey, make(map[string]any))
		platformConfig[connectionName] = remotePlatform
		settings[remotePlatformKey] = platformConfig
	}

	if missing.listenOnSocket {
		settings[listenOnSocketKey] = true
	}

	if len(missing.extensions) > 0 {
		extArray := getOrDefault(settings, defaultExtensionsKey, []any{})
		existing := make(map[string]bool)
		for _, ext := range extArray {
			if extStr, ok := ext.(string); ok {
				existing[extStr] = true
			}
		}
		for _, ext := range missing.extensions {
			if !existing[ext] {
				extArray = append(extArray, ext)
			}
		}
		settings[defaultExtensionsKey] = extArray
	}
}

func saveSettings(path string, settings map[string]any) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

func GetManualInstructions(ide, connectionName string) string {
	return fmt.Sprintf(
		"To ensure the remote connection works as expected, manually add these settings to your %s settings.json:\n"+
			"  \"%s\": {\"%s\": \"%s\"},\n"+
			"  \"%s\": {\"%s\": \"%s\"},\n"+
			"  \"%s\": true,\n"+
			"  \"%s\": [\"%s\", \"%s\"]",
		getIDEName(ide),
		serverPickPortsKey, connectionName, portRange,
		remotePlatformKey, connectionName, remotePlatform,
		listenOnSocketKey,
		defaultExtensionsKey, pythonExtension, jupyterExtension)
}
