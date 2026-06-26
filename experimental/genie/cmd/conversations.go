package geniecmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/env"
)

// Conversation labels are mapped to server conversation ids in
// ~/.databricks/genie-conversations.json, following the same ~/.databricks JSON
// store pattern as cmd/sandbox/state.go. The mapping is best-effort: any
// read/write error fails open and the ask just starts a fresh conversation.
const (
	conversationStorePath = ".databricks/genie-conversations.json"
	conversationDirPerm   = 0o700
	conversationFilePerm  = 0o600

	// conversationTTL bounds how long a label stays mapped. Genie conversations
	// are ephemeral server-side and a stale mapping self-heals, so it only needs
	// to be generous.
	conversationTTL = 14 * 24 * time.Hour
)

type conversationEntry struct {
	ID        string    `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
}

// conversationKey scopes a label by workspace host, so the same label on
// different workspaces never collides.
func conversationKey(host, label string) string {
	return host + "\n" + label
}

func conversationStoreLocation(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, conversationStorePath), nil
}

// loadConversations reads the store, dropping entries past the TTL. A missing or
// unreadable file yields an empty map. The store path is returned for a
// follow-up save (empty if it can't be resolved).
func loadConversations(ctx context.Context) (map[string]conversationEntry, string) {
	out := map[string]conversationEntry{}
	path, err := conversationStoreLocation(ctx)
	if err != nil {
		return out, ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, path
	}
	var stored map[string]conversationEntry
	if json.Unmarshal(raw, &stored) != nil {
		return out, path
	}
	for k, e := range stored {
		if time.Since(e.UpdatedAt) < conversationTTL {
			out[k] = e
		}
	}
	return out, path
}

func saveConversations(path string, m map[string]conversationEntry) {
	if path == "" {
		return
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), conversationDirPerm); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".genie-conversations-*.tmp")
	if err != nil {
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := tmp.Write(raw); err != nil {
		return
	}
	if err := tmp.Chmod(conversationFilePerm); err != nil {
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	_ = os.Rename(tmp.Name(), path)
}

// lookupConversationID returns the server conversation id mapped to label on
// host, or "" if there is no live mapping.
func lookupConversationID(ctx context.Context, host, label string) string {
	if label == "" {
		return ""
	}
	m, _ := loadConversations(ctx)
	return m[conversationKey(host, label)].ID
}

// storeConversationID records label -> serverID for later calls.
func storeConversationID(ctx context.Context, host, label, serverID string) {
	if label == "" || serverID == "" {
		return
	}
	m, path := loadConversations(ctx)
	m[conversationKey(host, label)] = conversationEntry{ID: serverID, UpdatedAt: time.Now()}
	saveConversations(path, m)
}

// forgetConversation drops the mapping for label so the next ask starts fresh.
// It is called when resuming fails because the server conversation is gone.
func forgetConversation(ctx context.Context, host, label string) {
	if label == "" {
		return
	}
	m, path := loadConversations(ctx)
	delete(m, conversationKey(host, label))
	saveConversations(path, m)
}
