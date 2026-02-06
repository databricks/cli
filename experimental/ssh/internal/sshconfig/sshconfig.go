package sshconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

const (
	// configDirName is the directory name for Databricks SSH tunnel configs, relative to the user's home directory.
	configDirName = ".databricks/ssh-tunnel-configs"
)

func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, configDirName), nil
}

func GetMainConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssh", "config"), nil
}

func GetMainConfigPathOrDefault(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return GetMainConfigPath()
}

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

	// Convert path to forward slashes for SSH config compatibility across platforms
	configDirUnix := filepath.ToSlash(configDir)

	includeLine := fmt.Sprintf("Include %s/*", configDirUnix)
	if strings.Contains(string(content), includeLine) {
		return nil
	}

	newContent := includeLine + "\n"
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

func GetHostConfigPath(hostName string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, hostName), nil
}

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

func PromptRecreateConfig(ctx context.Context, hostName string) (bool, error) {
	response, err := cmdio.AskYesOrNo(ctx, fmt.Sprintf("Host '%s' already exists. Do you want to recreate the config?", hostName))
	if err != nil {
		return false, err
	}
	return response, nil
}

func GenerateHostConfig(hostName, userName, identityFile, proxyCommand string) string {
	return fmt.Sprintf(`
Host %s
    User %s
    ConnectTimeout 360
    StrictHostKeyChecking accept-new
    IdentitiesOnly yes
    IdentityFile %q
    ProxyCommand %s
`, hostName, userName, identityFile, proxyCommand)
}
