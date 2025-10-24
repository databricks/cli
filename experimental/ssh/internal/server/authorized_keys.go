package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

type AuthorizedKeysManager struct {
	mu          sync.Mutex
	filePath    string
	client      *databricks.WorkspaceClient
	secretScope string
	addedKeys   map[string]bool
}

func NewAuthorizedKeysManager(client *databricks.WorkspaceClient, filePath, secretScope string) *AuthorizedKeysManager {
	return &AuthorizedKeysManager{
		filePath:    filePath,
		client:      client,
		secretScope: secretScope,
		addedKeys:   make(map[string]bool),
	}
}

// Adds a public key from secrets scope to the authorized_keys file.
// If the key has already been added, this is a no-op.
func (akm *AuthorizedKeysManager) AddKey(ctx context.Context, publicKeyName string) error {
	akm.mu.Lock()
	defer akm.mu.Unlock()

	if akm.addedKeys[publicKeyName] {
		log.Infof(ctx, "Public key %s already added, skipping", publicKeyName)
		return nil
	}

	log.Infof(ctx, "Adding public key from secret: %s", publicKeyName)
	clientPublicKey, err := keys.GetSecret(ctx, akm.client, akm.secretScope, publicKeyName)
	if err != nil {
		return fmt.Errorf("failed to get client public key: %w", err)
	}

	authKeys, err := os.OpenFile(akm.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open authorized keys file: %w", err)
	}
	defer authKeys.Close()

	content := strings.TrimSpace(string(clientPublicKey))
	_, err = authKeys.WriteString("\n" + content)
	if err != nil {
		return err
	}

	akm.addedKeys[publicKeyName] = true
	return nil
}
