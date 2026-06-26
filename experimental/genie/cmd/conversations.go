package geniecmd

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/cache"
)

// conversationTTL bounds how long a client-chosen label stays mapped to a server
// conversation id. Genie conversations are ephemeral server-side, and a stale
// mapping self-heals (a failed resume forgets the label, so the next ask starts
// fresh), so this only needs to be generous, not exact.
const conversationTTL = 14 * 24 * time.Hour

// conversationComponent is the libs/cache component name; it becomes a directory
// under the user cache dir (e.g. ~/.cache/databricks/<version>/genie-conversations).
const conversationComponent = "genie-conversations"

// conversationKey maps a client-chosen label to a server conversation id, scoped
// by workspace host so the same label on different workspaces never collides.
type conversationKey struct {
	Host  string `json:"host"`
	Label string `json:"label"`
}

func newConversationCache(ctx context.Context) *cache.Cache {
	return cache.NewCache(ctx, conversationComponent, conversationTTL, nil)
}

// lookupConversationID returns the server conversation id mapped to label on
// host, or "" if there is no (live) mapping.
func lookupConversationID(ctx context.Context, c *cache.Cache, host, label string) string {
	if label == "" {
		return ""
	}
	id, _ := cache.Get[string](ctx, c, conversationKey{Host: host, Label: label})
	return id
}

// storeConversationID records label -> serverID so a later ask with the same
// label continues the conversation.
func storeConversationID(ctx context.Context, c *cache.Cache, host, label, serverID string) {
	if label == "" || serverID == "" {
		return
	}
	cache.Put(ctx, c, conversationKey{Host: host, Label: label}, serverID)
}

// forgetConversation drops the mapping for label. It is called when resuming
// fails (the server conversation expired or is gone), so the next ask with the
// label starts a fresh conversation instead of retrying a dead id. The mapping
// is overwritten with an empty id, which lookupConversationID treats as absent.
func forgetConversation(ctx context.Context, c *cache.Cache, host, label string) {
	if label == "" {
		return
	}
	cache.Put(ctx, c, conversationKey{Host: host, Label: label}, "")
}
