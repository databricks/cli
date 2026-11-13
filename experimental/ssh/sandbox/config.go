package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/sshconfig"
)

// ValidateHostAlias restricts the alias to a literal host and a portable file name.
func ValidateHostAlias(alias string) error {
	if alias == "" || alias[0] == '-' || alias == "." || alias == ".." || strings.HasSuffix(alias, ".") {
		return fmt.Errorf("invalid SSH host alias %q", alias)
	}
	for _, character := range alias {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && !strings.ContainsRune("._-", character) {
			return errors.New("SSH host alias must contain only letters, numbers, dots, underscores or hyphens")
		}
	}
	stem := strings.ToUpper(strings.SplitN(alias, ".", 2)[0])
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return fmt.Errorf("SSH host alias %q is a reserved file name", alias)
	}
	return nil
}

// WriteHostConfig reuses compute SSH's Include directory and host-block writer.
// Existing aliases require an explicit overwrite unless the content is unchanged.
func WriteHostConfig(ctx context.Context, alias, mainConfig, proxyCommand string, identity *Identity, overwrite bool) (string, error) {
	if err := ValidateHostAlias(alias); err != nil {
		return "", err
	}
	config := sshconfig.GenerateHostConfigWithTimeout(alias, "sandbox-agent", escapeSSHPath(identity.KeyPath), escapeSSHPath(identity.KnownHostsPath), identity.HostKeyAlias, proxyCommand, 900)
	destination, err := sshconfig.GetHostConfigPath(ctx, alias)
	if err != nil {
		return "", err
	}
	existing, err := os.ReadFile(destination)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err == nil && string(existing) != config && !overwrite {
		return "", fmt.Errorf("SSH host alias %q already exists; choose another --name or pass --overwrite", alias)
	}
	mainConfig, err = sshconfig.GetMainConfigPathOrDefault(ctx, mainConfig)
	if err != nil {
		return "", err
	}
	if err := sshconfig.EnsureIncludeDirective(ctx, mainConfig); err != nil {
		return "", err
	}
	if string(existing) != config {
		if _, err := sshconfig.CreateOrUpdateHostConfig(ctx, alias, config, true); err != nil {
			return "", err
		}
	}
	return destination, nil
}

func escapeSSHPath(value string) string {
	return strings.ReplaceAll(filepath.ToSlash(value), "%", "%%")
}
