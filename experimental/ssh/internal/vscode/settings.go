package vscode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/fileutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/tailscale/hujson"
)

const (
	// TODO: change to 4000-4005 when the relevant changes are fully deployed.
	// The ports below can also be used by PyTorch, but we have a bigger range available for it.
	portRange            = "29500-29505"
	remotePlatform       = "linux"
	pythonExtension      = "ms-python.python"
	jupyterExtension     = "ms-toolsai.jupyter"
	databricksExtension  = "databricks.databricks"
	serverPickPortsKey   = "remote.SSH.serverPickPortsFromRange"
	remotePlatformKey    = "remote.SSH.remotePlatform"
	defaultExtensionsKey = "remote.SSH.defaultExtensions"
	listenOnSocketKey    = "remote.SSH.remoteServerListenOnSocket"
)

type missingSettings struct {
	portRange      bool
	platform       bool
	listenOnSocket bool
	extensions     []string
}

func (m *missingSettings) isEmpty() bool {
	return !m.portRange && !m.platform && !m.listenOnSocket && len(m.extensions) == 0
}

// Builds a JSON Pointer (RFC 6901) from path segments to be used in hujson.Value.Find.
// Escapes "~" → "~0" and "/" → "~1" per spec.
func jsonPtr(segments ...string) string {
	var b strings.Builder
	r := strings.NewReplacer("~", "~0", "/", "~1")
	for _, s := range segments {
		b.WriteByte('/')
		b.WriteString(r.Replace(s))
	}
	return b.String()
}

type patchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

func logSkippingSettings(ctx context.Context, msg string) {
	cmdio.LogString(ctx, msg+"\n\nWARNING: the connection might not work as expected\n")
}

func CheckAndUpdateSettings(ctx context.Context, ide, connectionName string) error {
	if !cmdio.IsPromptSupported(ctx) {
		logSkippingSettings(ctx, "Skipping IDE settings check: prompts not supported")
		return nil
	}

	settingsPath, err := getDefaultSettingsPath(ctx, ide)
	if err != nil {
		return fmt.Errorf("failed to get settings path: %w", err)
	}

	settings, err := loadSettings(settingsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
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
		logSkippingSettings(ctx, "Skipping IDE settings update")
		return nil
	}

	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := fileutil.BackupFile(ctx, settingsPath, data); err != nil {
			return fmt.Errorf("failed to backup settings: %w", err)
		}
	}

	if err := updateSettings(&settings, connectionName, missing); err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	if err := saveSettings(settingsPath, &settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Updated %s settings for '%s'", getIDE(ide).Name, connectionName))
	return nil
}

func getDefaultSettingsPath(ctx context.Context, ide string) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	appName := getIDE(ide).AppName

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

func loadSettings(path string) (hujson.Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return hujson.Value{}, err
	}
	v, err := hujson.Parse(data)
	if err != nil {
		return hujson.Value{}, fmt.Errorf("failed to parse settings JSON: %w", err)
	}
	return v, nil
}

func hasCorrectPortRange(v hujson.Value, connectionName string) bool {
	found := v.Find(jsonPtr(serverPickPortsKey, connectionName))
	if found == nil {
		return false
	}
	lit, ok := found.Value.(hujson.Literal)
	return ok && lit.String() == portRange
}

func hasCorrectPlatform(v hujson.Value, connectionName string) bool {
	found := v.Find(jsonPtr(remotePlatformKey, connectionName))
	if found == nil {
		return false
	}
	lit, ok := found.Value.(hujson.Literal)
	return ok && lit.String() == remotePlatform
}

func hasCorrectListenOnSocket(v hujson.Value) bool {
	found := v.Find(jsonPtr(listenOnSocketKey))
	if found == nil {
		return false
	}
	lit, ok := found.Value.(hujson.Literal)
	return ok && lit.Bool()
}

func getMissingExtensions(v hujson.Value) []string {
	required := []string{pythonExtension, jupyterExtension, databricksExtension}
	found := v.Find(jsonPtr(defaultExtensionsKey))
	if found == nil {
		return required
	}
	arr, ok := found.Value.(*hujson.Array)
	if !ok {
		return required
	}
	existingSet := make(map[string]bool, len(arr.Elements))
	for _, el := range arr.Elements {
		if lit, ok := el.Value.(hujson.Literal); ok {
			existingSet[lit.String()] = true
		}
	}
	var missing []string
	for _, ext := range required {
		if !existingSet[ext] {
			missing = append(missing, ext)
		}
	}
	return missing
}

func validateSettings(v hujson.Value, connectionName string) *missingSettings {
	return &missingSettings{
		portRange:      !hasCorrectPortRange(v, connectionName),
		platform:       !hasCorrectPlatform(v, connectionName),
		listenOnSocket: !hasCorrectListenOnSocket(v),
		extensions:     getMissingExtensions(v),
	}
}

func settingsMessage(connectionName string, missing *missingSettings) string {
	var lines []string
	if missing.portRange {
		lines = append(lines, fmt.Sprintf("    \"%s\": {\"%s\": \"%s\"}", serverPickPortsKey, connectionName, portRange))
	}
	if missing.platform {
		lines = append(lines, fmt.Sprintf("    \"%s\": {\"%s\": \"%s\"}", remotePlatformKey, connectionName, remotePlatform))
	}
	if missing.listenOnSocket {
		lines = append(lines, fmt.Sprintf("    \"%s\": true // Global setting", listenOnSocketKey))
	}
	if len(missing.extensions) > 0 {
		quoted := make([]string, len(missing.extensions))
		for i, ext := range missing.extensions {
			quoted[i] = fmt.Sprintf("\"%s\"", ext)
		}
		lines = append(lines, fmt.Sprintf("    \"%s\": [%s] // Global setting", defaultExtensionsKey, strings.Join(quoted, ", ")))
	}
	return "  {\n" + strings.Join(lines, ",\n") + "\n  }"
}

func promptUserForUpdate(ctx context.Context, ide, connectionName string, missing *missingSettings) (bool, error) {
	question := fmt.Sprintf(
		"The following settings will be applied to %s for '%s':\n\n%s\n\nApply these settings?",
		getIDE(ide).Name, connectionName, settingsMessage(connectionName, missing))
	ans, err := cmdio.Ask(ctx, question+" [Y/n]", "y")
	if err != nil {
		return false, err
	}
	return strings.ToLower(ans) == "y", nil
}

func handleMissingFile(ctx context.Context, ide, connectionName, settingsPath string) error {
	missing := &missingSettings{
		portRange:      true,
		platform:       true,
		listenOnSocket: true,
		extensions:     []string{pythonExtension, jupyterExtension, databricksExtension},
	}
	shouldCreate, err := promptUserForUpdate(ctx, ide, connectionName, missing)
	if err != nil {
		return fmt.Errorf("failed to prompt user: %w", err)
	}
	if !shouldCreate {
		logSkippingSettings(ctx, "Skipping IDE settings creation")
		return nil
	}

	settingsDir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	v, err := hujson.Parse([]byte("{}"))
	if err != nil {
		return fmt.Errorf("failed to create settings: %w", err)
	}
	if err := updateSettings(&v, connectionName, missing); err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	if err := saveSettings(settingsPath, &v); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Created %s settings at %s", getIDE(ide).Name, filepath.ToSlash(settingsPath)))
	return nil
}

// subKeyOp returns a patch op that sets key/subKey=value, creating the parent object if absent.
func subKeyOp(v *hujson.Value, key, subKey, value string) patchOp {
	if v.Find(jsonPtr(key)) == nil {
		return patchOp{"add", jsonPtr(key), map[string]string{subKey: value}}
	}
	return patchOp{"add", jsonPtr(key, subKey), value}
}

func updateSettings(v *hujson.Value, connectionName string, missing *missingSettings) error {
	var ops []patchOp
	if missing.portRange {
		ops = append(ops, subKeyOp(v, serverPickPortsKey, connectionName, portRange))
	}
	if missing.platform {
		ops = append(ops, subKeyOp(v, remotePlatformKey, connectionName, remotePlatform))
	}
	if missing.listenOnSocket {
		ops = append(ops, patchOp{"add", jsonPtr(listenOnSocketKey), true})
	}
	if len(missing.extensions) > 0 {
		parent := jsonPtr(defaultExtensionsKey)
		if v.Find(parent) == nil {
			ops = append(ops, patchOp{"add", parent, missing.extensions})
		} else {
			for _, ext := range missing.extensions {
				ops = append(ops, patchOp{"add", parent + "/-", ext})
			}
		}
	}
	if len(ops) == 0 {
		return nil
	}
	patchData, err := json.Marshal(ops)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}
	return v.Patch(patchData)
}

func saveSettings(path string, v *hujson.Value) error {
	if err := os.WriteFile(path, v.Pack(), 0o600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}
	return nil
}

func GetManualInstructions(ide, connectionName string) string {
	missing := &missingSettings{
		portRange:      true,
		platform:       true,
		listenOnSocket: true,
		extensions:     []string{pythonExtension, jupyterExtension, databricksExtension},
	}
	return fmt.Sprintf(
		"To ensure the remote connection works as expected, manually add these settings to your %s settings.json:\n%s",
		getIDE(ide).Name, settingsMessage(connectionName, missing))
}
