package sshconfig

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/fileutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
)

const (
	// configDirName is the directory name for Databricks SSH tunnel configs, relative to the user's home directory.
	configDirName = ".databricks/ssh-tunnel-configs"
)

// ServerAliveIntervalSeconds is how often the ssh client asks the SSH server to confirm it is
// still there. The reply is a real SSH packet, so the keepalive puts payload bytes on every hop
// of the tunnel — and payload is what an idle session needs. The driver proxy terminates
// websocket control frames itself, so the proxy's own websocket ping never becomes payload on
// the leg past it, and that leg is reaped after ~8 minutes without any (see DECO-28186). Setting
// this option by hand is the workaround the reporting customer verified over ~2 hours idle, and
// 30s is the value measured to hold an otherwise-idle tunnel open, with wide margin under the
// reap window.
//
// It also brings in ServerAliveCountMax (OpenSSH default 3), so a tunnel that stops responding
// is torn down after ~90s with ssh's own "server not responding" message instead of hanging.
// That is well clear of the up to 30s a handover can hold the sending loop
// (proxyHandoverInitTimeout), the longest legitimate pause on a healthy tunnel.
const ServerAliveIntervalSeconds = 30

func GetConfigDir(ctx context.Context) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, configDirName), nil
}

func GetMainConfigPath(ctx context.Context) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ssh", "config"), nil
}

func GetMainConfigPathOrDefault(ctx context.Context, configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return GetMainConfigPath(ctx)
}

func EnsureMainConfigExists(configPath string) error {
	_, err := os.Stat(configPath)
	if errors.Is(err, fs.ErrNotExist) {
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

func EnsureIncludeDirective(ctx context.Context, configPath string) error {
	configDir, err := GetConfigDir(ctx)
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

	// Quoted to handle paths with spaces; OpenSSH still expands globs inside quotes.
	includeLine := fmt.Sprintf(`Include "%s/*"`, configDirUnix)
	if containsLine(content, includeLine) {
		return nil
	}

	// Migrate unquoted Include written by older versions of the CLI.
	oldIncludeLine := fmt.Sprintf("Include %s/*", configDirUnix)
	if containsLine(content, oldIncludeLine) {
		if err := fileutil.BackupFile(ctx, configPath, content); err != nil {
			return fmt.Errorf("failed to backup SSH config before migration: %w", err)
		}
		return os.WriteFile(configPath, replaceLine(content, oldIncludeLine, includeLine), 0o600)
	}

	if err := fileutil.BackupFile(ctx, configPath, content); err != nil {
		return fmt.Errorf("failed to backup SSH config: %w", err)
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

// containsLine reports whether data contains line as a line match,
// trimming leading whitespace and \r (Windows line endings) before comparing.
func containsLine(data []byte, line string) bool {
	for l := range strings.SplitSeq(string(data), "\n") {
		if strings.TrimLeft(strings.TrimRight(l, "\r"), " \t") == line {
			return true
		}
	}
	return false
}

// replaceLine replaces the first line in data whose trimmed content matches old
// with new. Uses the same trim logic as containsLine. Returns data unchanged if
// no match.
func replaceLine(data []byte, old, new string) []byte {
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		if strings.TrimLeft(strings.TrimRight(l, "\r"), " \t") == old {
			lines[i] = new
			break
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func GetHostConfigPath(ctx context.Context, hostName string) (string, error) {
	configDir, err := GetConfigDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, hostName), nil
}

func HostConfigExists(ctx context.Context, hostName string) (bool, error) {
	configPath, err := GetHostConfigPath(ctx, hostName)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(configPath)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check host config file: %w", err)
	}
	return true, nil
}

// Returns true if the config was created/updated, false if it was skipped.
func CreateOrUpdateHostConfig(ctx context.Context, hostName, hostConfig string, recreate bool) (bool, error) {
	configPath, err := GetHostConfigPath(ctx, hostName)
	if err != nil {
		return false, err
	}

	exists, err := HostConfigExists(ctx, hostName)
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
    ServerAliveInterval %d
    StrictHostKeyChecking accept-new
    IdentitiesOnly yes
    IdentityFile %q
    ProxyCommand %s
`, hostName, userName, ServerAliveIntervalSeconds, identityFile, proxyCommand)
}
