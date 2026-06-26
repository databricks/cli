package geniecmd

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestConversationStore(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	const host = "https://example.cloud.databricks.com"

	// Miss before anything is stored, and an empty label never maps.
	assert.Empty(t, lookupConversationID(ctx, host, "q3"))
	assert.Empty(t, lookupConversationID(ctx, host, ""))

	// Store then resolve.
	storeConversationID(ctx, host, "q3", "server-1")
	assert.Equal(t, "server-1", lookupConversationID(ctx, host, "q3"))

	// Labels are scoped by host, and distinct labels are independent.
	assert.Empty(t, lookupConversationID(ctx, "https://other.databricks.com", "q3"))
	assert.Empty(t, lookupConversationID(ctx, host, "q4"))

	// Forgetting a stale mapping makes the label resolve fresh again.
	forgetConversation(ctx, host, "q3")
	assert.Empty(t, lookupConversationID(ctx, host, "q3"))

	// Entries past the TTL are ignored.
	path, _ := conversationStoreLocation(ctx)
	saveConversations(path, map[string]conversationEntry{
		conversationKey(host, "old"): {ID: "server-2", UpdatedAt: time.Now().Add(-2 * conversationTTL)},
	})
	assert.Empty(t, lookupConversationID(ctx, host, "old"))
}
