package keys

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/crypto/ssh"
)

// We use different client keys for each cluster as a good practice for better isolation and control.
func GetLocalSSHKeyPath(clusterID, keysDir string) (string, error) {
	if keysDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		keysDir = filepath.Join(homeDir, ".databricks", "ssh-tunnel-keys")
	}
	return filepath.Join(keysDir, clusterID), nil
}

func generateSSHKeyPair() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return privateKeyPEM, publicKeyBytes, nil
}

func SaveSSHKeyPair(keyPath string, privateKeyBytes, publicKeyBytes []byte) error {
	err := os.RemoveAll(filepath.Dir(keyPath))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing key directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return fmt.Errorf("failed to create directory for key: %w", err)
	}

	if err := os.WriteFile(keyPath, privateKeyBytes, 0o600); err != nil {
		return fmt.Errorf("failed to write private key to file: %w", err)
	}

	if err := os.WriteFile(keyPath+".pub", publicKeyBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write public key to file: %w", err)
	}

	return nil
}

func checkSSHKeyPairPermissions(keyPath string) error {
	dir := filepath.Dir(keyPath)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key directory does not exist: %s", dir)
		} else {
			return fmt.Errorf("failed to stat key directory: %w", err)
		}
	}
	if runtime.GOOS != "windows" && dirInfo.Mode().Perm() != 0o700 {
		return fmt.Errorf("key directory permissions are not set to 0700, current permissions: %o", dirInfo.Mode().Perm())
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("failed to stat key file: %w", err)
	}

	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		return fmt.Errorf("private key permissions are not set to 0600, current permissions: %o", info.Mode().Perm())
	}

	pubKeyPath := keyPath + ".pub"
	pubInfo, err := os.Stat(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to stat public key file: %w", err)
	}

	if runtime.GOOS != "windows" && pubInfo.Mode().Perm() != 0o644 {
		return fmt.Errorf("public key permissions are not set to 0644, current permissions: %o", pubInfo.Mode().Perm())
	}

	return nil
}

func CheckAndGenerateSSHKeyPair(ctx context.Context, keyPath string) (string, string, error) {
	if err := checkSSHKeyPairPermissions(keyPath); err != nil {
		privateKeyBytes, publicKeyBytes, err := generateSSHKeyPair()
		if err != nil {
			return "", "", err
		}
		if err := SaveSSHKeyPair(keyPath, privateKeyBytes, publicKeyBytes); err != nil {
			return "", "", err
		}
	}

	publicKeyBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", "", fmt.Errorf("failed to read public key: %w", err)
	}

	return keyPath, strings.TrimSpace(string(publicKeyBytes)), nil
}

func CheckAndGenerateSSHKeyPairFromSecrets(ctx context.Context, client *databricks.WorkspaceClient, clusterID, secretsScopeName, privateKeyName, publicKeyName string) ([]byte, []byte, error) {
	privateKeyBytes, err := GetSecret(ctx, client, secretsScopeName, privateKeyName)
	if err != nil {
		privateKeyBytes, publicKeyBytes, err := generateSSHKeyPair()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate SSH key pair: %w", err)
		}

		err = putSecret(ctx, client, secretsScopeName, privateKeyName, string(privateKeyBytes))
		if err != nil {
			return nil, nil, err
		}

		err = putSecret(ctx, client, secretsScopeName, publicKeyName, string(publicKeyBytes))
		if err != nil {
			return nil, nil, err
		}

		return privateKeyBytes, publicKeyBytes, nil
	} else {
		publicKeyBytes, err := GetSecret(ctx, client, secretsScopeName, publicKeyName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get public key from secrets scope: %w", err)
		}

		return privateKeyBytes, publicKeyBytes, nil
	}
}
