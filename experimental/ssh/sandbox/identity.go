package sandbox

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/experimental/ssh/internal/sshconfig"
	"golang.org/x/crypto/ssh"
)

// Identity contains only paths and public material; private keys remain local.
type Identity struct {
	KeyPath        string
	KnownHostsPath string
	HostKeyAlias   string
	PublicKey      []byte
}

// LocalIdentity creates a reusable key scoped to the workspace, caller and sandbox.
func LocalIdentity(ctx context.Context, workspace, caller, name string) (*Identity, error) {
	scope := fmt.Sprintf("sandbox-%x", sha256.Sum256([]byte(workspace+"\x00"+caller+"\x00"+name)))
	directory, err := keys.GetLocalSSHKeyPath(ctx, scope, "")
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(directory, "id_ed25519")
	if _, err := os.Stat(directory); errors.Is(err, fs.ErrNotExist) {
		if err := createLocalKey(directory); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	private, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read local SSH key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(private)
	if err != nil {
		return nil, fmt.Errorf("parse local SSH key: %w", err)
	}
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return nil, err
	}
	knownHosts, err := sshconfig.GetKnownHostsPath(ctx, scope, "")
	if err != nil {
		return nil, err
	}
	return &Identity{KeyPath: keyPath, KnownHostsPath: knownHosts, HostKeyAlias: scope, PublicKey: ssh.MarshalAuthorizedKey(signer.PublicKey())}, nil
}

func createLocalKey(directory string) error {
	if err := os.MkdirAll(filepath.Dir(directory), 0o700); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(filepath.Dir(directory), ".sandbox-key-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	encoded, err := ssh.MarshalPrivateKey(private, "")
	if err != nil {
		return err
	}
	public, err := ssh.NewPublicKey(private.Public())
	if err != nil {
		return err
	}
	if err := keys.SaveSSHKeyPair(filepath.Join(temporary, "id_ed25519"), pem.EncodeToMemory(encoded), ssh.MarshalAuthorizedKey(public)); err != nil {
		return err
	}
	// Rename a complete directory: concurrent first connections either install
	// their key or read the winner, without ever replacing an in-use private key.
	if err := os.Rename(temporary, directory); err != nil {
		if _, readErr := os.Stat(filepath.Join(directory, "id_ed25519")); readErr != nil {
			return fmt.Errorf("install local SSH key: %w", err)
		}
	}
	return nil
}

func (identity *Identity) PinHostKey(key []byte) error {
	return sshconfig.PinHostKey(identity.KnownHostsPath, identity.HostKeyAlias, key)
}
