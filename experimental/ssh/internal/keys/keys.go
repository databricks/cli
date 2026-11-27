package keys

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/crypto/ssh"
)

// We use different client keys for each session as a good practice for better isolation and control.
// sessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
func GetLocalSSHKeyPath(ctx context.Context, sessionID, keysDir string) (string, error) {
	if keysDir == "" {
		homeDir, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		keysDir = filepath.Join(homeDir, ".databricks", "ssh-tunnel-keys")
	}
	return filepath.Join(keysDir, sessionID), nil
}

// GetLocalServerSSHKeyPath returns where the client key pair of one server of a session is
// saved, the server that authorized the client key with clientKeyFingerprint. It sits next to
// the session's key (see GetLocalSSHKeyPath), named after the session and a digest of the
// fingerprint; session IDs contain no dot, so it cannot be another session's key.
//
// Servers started for a session at the same time each authorize a key pair of their own, and
// the ssh client or IDE configuration a connection sets up for one of them must not find the
// other's key in the file it reads, so a connect to another server of the session never
// rewrites the file. A ProxyCommand that names the server does when it finds the server
// replaced on the same port and the file missing: ssh has loaded no key from it then, so it
// saves the pair the replacement authorized there.
func GetLocalServerSSHKeyPath(ctx context.Context, sessionID, clientKeyFingerprint, keysDir string) (string, error) {
	sessionPath, err := GetLocalSSHKeyPath(ctx, sessionID, keysDir)
	if err != nil {
		return "", err
	}
	return sessionPath + FingerprintFileSuffix(clientKeyFingerprint), nil
}

// FingerprintFileSuffix returns the suffix that names the file of one server of a session
// after a key fingerprint: a dot and a digest of the fingerprint. Both the client key files
// (see GetLocalServerSSHKeyPath) and the known_hosts files of a server use it.
func FingerprintFileSuffix(fingerprint string) string {
	digest := sha256.Sum256([]byte(fingerprint))
	return "." + hex.EncodeToString(digest[:8])
}

// RemoveOtherServerFiles removes the files of the session's other servers that sit next to
// keepPath, the file of the server a connect picked: the files GetLocalServerSSHKeyPath names
// with their .pub files, or the known_hosts files sshconfig.GetServerKnownHostsPath names.
// Each server start leaves new ones, so they would otherwise pile up. The session's own file,
// which ssh configurations written by `databricks ssh setup` read, is kept.
func RemoveOtherServerFiles(sessionID, keepPath string) error {
	dir, keepName := filepath.Split(keepPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	serverFile := regexp.MustCompile(`^` + regexp.QuoteMeta(sessionID) + `\.[0-9a-f]{16}(\.pub)?$`)
	var errs []error
	for _, entry := range entries {
		name := entry.Name()
		if !serverFile.MatchString(name) || name == keepName || name == keepName+".pub" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// GenerateSSHKeyPair generates a new key pair, returned as a PEM private key and an
// authorized_keys public key.
func GenerateSSHKeyPair() ([]byte, []byte, error) {
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

// ErrInvalidPrivateKey is returned by LoadSSHKeyPair when the file holds no private key it can
// parse, for example one cut short while it was written.
var ErrInvalidPrivateKey = errors.New("invalid SSH private key")

// LoadSSHKeyPair reads the private key SaveSSHKeyPair wrote at keyPath and returns it with
// its public key. The public key is derived from the private one rather than read from the
// .pub file, so it carries no authorized_keys options and always matches the private key.
// The error wraps fs.ErrNotExist when there is no key at keyPath, and ErrInvalidPrivateKey
// when the file does not parse.
func LoadSSHKeyPair(keyPath string) ([]byte, []byte, error) {
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to parse private key %s: %w", ErrInvalidPrivateKey, keyPath, err)
	}
	return privateKeyBytes, ssh.MarshalAuthorizedKey(signer.PublicKey()), nil
}

// PublicKeyFingerprint returns the SHA256 fingerprint of an authorized_keys public key, in the
// form ssh-keygen -l prints it.
func PublicKeyFingerprint(publicKey []byte) (string, error) {
	parsed, _, _, _, err := ssh.ParseAuthorizedKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}
	return ssh.FingerprintSHA256(parsed), nil
}

// PutSSHKeyPairToSecrets stores a key pair in the secret scope, overwriting whatever the
// two secrets held before.
func PutSSHKeyPairToSecrets(ctx context.Context, client *databricks.WorkspaceClient, secretScopeName, privateKeyName, publicKeyName string, privateKeyBytes, publicKeyBytes []byte) error {
	err := putSecret(ctx, client, secretScopeName, privateKeyName, string(privateKeyBytes))
	if err != nil {
		return err
	}
	return putSecret(ctx, client, secretScopeName, publicKeyName, string(publicKeyBytes))
}

// GenerateSSHKeyPairToSecrets generates a new key pair and stores it in the secret scope,
// without reading what the scope held before.
func GenerateSSHKeyPairToSecrets(ctx context.Context, client *databricks.WorkspaceClient, secretScopeName, privateKeyName, publicKeyName string) ([]byte, []byte, error) {
	privateKeyBytes, publicKeyBytes, err := GenerateSSHKeyPair()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate SSH key pair: %w", err)
	}
	err = PutSSHKeyPairToSecrets(ctx, client, secretScopeName, privateKeyName, publicKeyName, privateKeyBytes, publicKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	return privateKeyBytes, publicKeyBytes, nil
}

// ErrMismatchedKeyPair is returned by GetSSHKeyPairFromSecrets when the two secrets hold
// halves of different key pairs.
var ErrMismatchedKeyPair = errors.New("hold halves of different key pairs")

// GetSSHKeyPairFromSecrets reads a key pair from the secret scope. It fails if either half
// is missing, or with ErrMismatchedKeyPair if the halves belong to different pairs:
// PutSSHKeyPairToSecrets writes them in two requests, so a failed second write or two writers
// interleaving can leave them mismatched.
func GetSSHKeyPairFromSecrets(ctx context.Context, client *databricks.WorkspaceClient, secretScopeName, privateKeyName, publicKeyName string) ([]byte, []byte, error) {
	privateKeyBytes, err := GetSecret(ctx, client, secretScopeName, privateKeyName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get private key from secrets scope: %w", err)
	}
	publicKeyBytes, err := GetSecret(ctx, client, secretScopeName, publicKeyName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get public key from secrets scope: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse secret %s of secret scope %s as a private key: %w", privateKeyName, secretScopeName, err)
	}
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(publicKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse secret %s of secret scope %s as a public key: %w", publicKeyName, secretScopeName, err)
	}
	if !bytes.Equal(publicKey.Marshal(), signer.PublicKey().Marshal()) {
		return nil, nil, fmt.Errorf("secrets %s and %s of secret scope %s %w, "+
			"which happens when a write of the pair failed halfway or two writes interleaved", privateKeyName, publicKeyName, secretScopeName, ErrMismatchedKeyPair)
	}
	return privateKeyBytes, ssh.MarshalAuthorizedKey(signer.PublicKey()), nil
}
