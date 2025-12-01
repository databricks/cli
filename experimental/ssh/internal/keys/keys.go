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

func CheckAndGenerateSSHKeyPairFromSecrets(ctx context.Context, client *databricks.WorkspaceClient, clusterID, secretScopeName, privateKeyName, publicKeyName string) ([]byte, []byte, error) {
	privateKeyBytes, err := GetSecret(ctx, client, secretScopeName, privateKeyName)
	if err != nil {
		privateKeyBytes, publicKeyBytes, err := generateSSHKeyPair()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate SSH key pair: %w", err)
		}

		err = putSecret(ctx, client, secretScopeName, privateKeyName, string(privateKeyBytes))
		if err != nil {
			return nil, nil, err
		}

		err = putSecret(ctx, client, secretScopeName, publicKeyName, string(publicKeyBytes))
		if err != nil {
			return nil, nil, err
		}

		return privateKeyBytes, publicKeyBytes, nil
	} else {
		publicKeyBytes, err := GetSecret(ctx, client, secretScopeName, publicKeyName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get public key from secrets scope: %w", err)
		}

		return privateKeyBytes, publicKeyBytes, nil
	}
}
