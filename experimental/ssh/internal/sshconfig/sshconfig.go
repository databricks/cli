package sshconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

const (
	// ConfigDirName is the directory name for Databricks SSH tunnel configs
	ConfigDirName = ".databricks/ssh-tunnel-configs"
)

// GetConfigDir returns the path to the Databricks SSH tunnel configs directory.
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDirName), nil
}

// GetMainConfigPath returns the path to the main SSH config file.
func GetMainConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssh", "config"), nil
}

// GetMainConfigPathOrDefault returns the provided path if non-empty, otherwise returns the default.
func GetMainConfigPathOrDefault(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return GetMainConfigPath()
}

// EnsureMainConfigExists ensures the main SSH config file exists.
func EnsureMainConfigExists(configPath string) error {
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		sshDir := filepath.Dir(configPath)
		err = os.MkdirAll(sshDir, 0o700)
		if err != nil {
			return fmt.Errorf("failed to create SSH directory: %w", err)
		}
		err = os.WriteFile(configPath, []byte(""), 0o600)
		if err != nil {
			return fmt.Errorf("failed to create SSH config file: %w", err)
		}
		return nil
	}
	return err
}

// EnsureIncludeDirective ensures the Include directive for Databricks configs exists in the main SSH config.
func EnsureIncludeDirective(configPath string) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	err = os.MkdirAll(configDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create Databricks SSH config directory: %w", err)
	}

	err = EnsureMainConfigExists(configPath)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config file: %w", err)
	}

	// Check if Include directive already exists
	includePattern := fmt.Sprintf(`(?m)^\s*Include\s+.*%s/\*\s*$`, regexp.QuoteMeta(ConfigDirName))
	matched, err := regexp.Match(includePattern, content)
	if err != nil {
		return fmt.Errorf("failed to check for existing Include directive: %w", err)
	}

	if matched {
		return nil
	}

	// Prepend the Include directive
	includeLine := fmt.Sprintf("Include %s/*\n", configDir)
	newContent := includeLine
	if len(content) > 0 && !strings.HasPrefix(string(content), "\n") {
		newContent += "\n"
	}
	newContent += string(content)

	err = os.WriteFile(configPath, []byte(newContent), 0o600)
	if err != nil {
		return fmt.Errorf("failed to update SSH config file with Include directive: %w", err)
	}

	return nil
}

// GetHostConfigPath returns the path to a specific host's config file.
func GetHostConfigPath(hostName string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, hostName), nil
}

// HostConfigExists checks if a host config file already exists.
func HostConfigExists(hostName string) (bool, error) {
	configPath, err := GetHostConfigPath(hostName)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check host config file: %w", err)
	}
	return true, nil
}

// CreateOrUpdateHostConfig creates or updates a host config file.
// Returns true if the config was created/updated, false if it was skipped.
func CreateOrUpdateHostConfig(ctx context.Context, hostName, hostConfig string, recreate bool) (bool, error) {
	configPath, err := GetHostConfigPath(hostName)
	if err != nil {
		return false, err
	}

	exists, err := HostConfigExists(hostName)
	if err != nil {
		return false, err
	}

	if exists && !recreate {
		return false, nil
	}

	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	err = os.MkdirAll(configDir, 0o700)
	if err != nil {
		return false, fmt.Errorf("failed to create config directory: %w", err)
	}

	err = os.WriteFile(configPath, []byte(hostConfig), 0o600)
	if err != nil {
		return false, fmt.Errorf("failed to write host config file: %w", err)
	}

	return true, nil
}

// PromptRecreateConfig asks the user if they want to recreate an existing config.
func PromptRecreateConfig(ctx context.Context, hostName string) (bool, error) {
	response, err := cmdio.AskYesOrNo(ctx, fmt.Sprintf("Host '%s' already exists. Do you want to recreate the config?", hostName))
	if err != nil {
		return false, err
	}
	return response, nil
}
