package ssh

import (
	"fmt"
	"os"
	"path/filepath"
)

func getLocalSSHKeyPath(clusterID, keysDir string) (string, error) {
	if keysDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		keysDir = filepath.Join(homeDir, ".databricks", "ssh-tunnel-keys")
	}
	return filepath.Join(keysDir, clusterID), nil
}
