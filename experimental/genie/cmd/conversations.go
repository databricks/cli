package geniecmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/env"
)

// A Genie conversation can be continued across CLI calls by giving it a session
// id (the --session flag). The server only accepts conversation ids it issued,
// so we map each session id to its server conversation id on disk, in
// ~/.databricks/genie-conversations.json (the same ~/.databricks JSON-store
// pattern as cmd/sandbox/state.go). The store is best-effort: any read/write
// error fails open and the ask just starts a fresh conversation.
const (
	// conversationStoreName is the store file inside ~/.databricks.
	conversationStoreName = "genie-conversations.json"
	conversationDirPerm   = 0o700
	conversationFilePerm  = 0o600

	// conversationTTL is how long a session mapping is kept before it expires.
	conversationTTL = 14 * 24 * time.Hour
)

// conversationEntry is the stored server conversation id for one session id,
// plus when it was last used (for TTL expiry).
type conversationEntry struct {
	ConversationID string    `json:"conversation_id"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// conversationStore maps workspace host -> session id -> entry. Scoping by host
// keeps the same session id on different workspaces from colliding.
type conversationStore map[string]map[string]conversationEntry

// loadStore reads and parses the on-disk store so a caller can look up or update
// a session mapping, dropping entries past the TTL. A missing or unreadable file
// yields an empty store. The resolved file path is returned so a follow-up
// saveStore knows where to write (empty if the home dir can't be resolved).
func loadStore(ctx context.Context) (conversationStore, string) {
	store := conversationStore{}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return store, ""
	}
	// Split path elements (not ".databricks/genie-..."), so the separator is
	// correct on Windows too.
	path := filepath.Join(home, ".databricks", conversationStoreName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return store, path
	}
	var stored conversationStore
	if json.Unmarshal(raw, &stored) != nil {
		return store, path
	}
	for host, sessions := range stored {
		for sessionID, e := range sessions {
			if time.Since(e.UpdatedAt) < conversationTTL {
				if store[host] == nil {
					store[host] = map[string]conversationEntry{}
				}
				store[host][sessionID] = e
			}
		}
	}
	return store, path
}

// saveStore writes the store back to path. It writes a temp file in the same
// directory and renames it into place, so a crash or concurrent reader never
// sees a half-written file. Errors are ignored: the store is best-effort.
func saveStore(path string, store conversationStore) {
	if path == "" {
		return
	}
	raw, err := json.MarshalIndent(store, "", "  ")
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

// lookupConversationID returns the server conversation id mapped to sessionID on
// host, or "" if there is no live mapping.
func lookupConversationID(ctx context.Context, host, sessionID string) string {
	if sessionID == "" {
		return ""
	}
	store, _ := loadStore(ctx)
	return store[host][sessionID].ConversationID
}

// storeConversationID records sessionID -> conversationID so a later call with
// the same session id continues the conversation.
func storeConversationID(ctx context.Context, host, sessionID, conversationID string) {
	if sessionID == "" || conversationID == "" {
		return
	}
	store, path := loadStore(ctx)
	if store[host] == nil {
		store[host] = map[string]conversationEntry{}
	}
	store[host][sessionID] = conversationEntry{ConversationID: conversationID, UpdatedAt: time.Now()}
	saveStore(path, store)
}

// forgetConversation drops the mapping for sessionID so the next ask starts
// fresh. It's called when resuming fails because the server conversation is gone.
func forgetConversation(ctx context.Context, host, sessionID string) {
	if sessionID == "" {
		return
	}
	store, path := loadStore(ctx)
	delete(store[host], sessionID)
	saveStore(path, store)
}
