package sshconfig

import (
	"context"
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
)

const (
	// configDirName is the directory name for Databricks SSH tunnel configs, relative to the user's home directory.
	configDirName = ".databricks/ssh-tunnel-configs"

	// socketsDirName is the directory name for SSH ControlMaster sockets, relative to the user's home directory.
	socketsDirName = ".databricks/ssh-sockets"
)

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

// GetSocketsDir returns the directory for SSH ControlMaster sockets.
func GetSocketsDir(ctx context.Context) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, socketsDirName), nil
}

// EnsureSocketsDir creates the ControlMaster sockets directory if it does not exist.
func EnsureSocketsDir(ctx context.Context) error {
	socketsDir, err := GetSocketsDir(ctx)
	if err != nil {
		return err
	}
	err = os.MkdirAll(socketsDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create SSH sockets directory: %w", err)
	}
	return nil
}

// HostConfigOptions contains the parameters for generating an SSH host config entry.
type HostConfigOptions struct {
	HostName     string
	UserName     string
	IdentityFile string
	ProxyCommand string
	// ControlPath enables SSH ControlMaster multiplexing when non-empty.
	// Ignored on Windows where ControlMaster is not supported.
	ControlPath string
}

// GenerateHostConfig generates an SSH host config entry from the given options.
func GenerateHostConfig(opts HostConfigOptions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nHost %s\n", opts.HostName)
	fmt.Fprintf(&b, "    User %s\n", opts.UserName)
	b.WriteString("    ConnectTimeout 360\n")
	b.WriteString("    StrictHostKeyChecking accept-new\n")
	b.WriteString("    IdentitiesOnly yes\n")
	fmt.Fprintf(&b, "    IdentityFile %q\n", opts.IdentityFile)
	fmt.Fprintf(&b, "    ProxyCommand %s\n", opts.ProxyCommand)

	if opts.ControlPath != "" && runtime.GOOS != "windows" {
		b.WriteString("    ControlMaster auto\n")
		fmt.Fprintf(&b, "    ControlPath %s\n", opts.ControlPath)
		b.WriteString("    ControlPersist 10m\n")
	}

	return b.String()
}
