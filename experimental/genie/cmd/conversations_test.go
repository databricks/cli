package geniecmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversationStore(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	const host = "https://example.cloud.databricks.com"

	// Miss before anything is stored, and an empty session id never maps.
	assert.Empty(t, lookupConversationID(ctx, host, "sales"))
	assert.Empty(t, lookupConversationID(ctx, host, ""))

	// Store then resolve.
	storeConversationID(ctx, host, "sales", "conv-1")
	assert.Equal(t, "conv-1", lookupConversationID(ctx, host, "sales"))

	// Session ids are scoped by host, and distinct session ids are independent.
	assert.Empty(t, lookupConversationID(ctx, "https://other.databricks.com", "sales"))
	assert.Empty(t, lookupConversationID(ctx, host, "revenue"))

	// Forgetting a stale mapping makes the session id resolve fresh again.
	forgetConversation(ctx, host, "sales")
	assert.Empty(t, lookupConversationID(ctx, host, "sales"))

	// Entries past the TTL are ignored.
	_, path := loadStore(ctx)
	saveStore(path, conversationStore{
		host: {"old": {ConversationID: "conv-2", UpdatedAt: time.Now().Add(-2 * conversationTTL)}},
	})
	assert.Empty(t, lookupConversationID(ctx, host, "old"))
}

func TestConversationStoreRecoversFromCorruptFile(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	const host = "https://example.cloud.databricks.com"

	// A malformed store file reads as empty (no error/panic)...
	_, path := loadStore(ctx)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("not valid json {{{"), 0o600))
	assert.Empty(t, lookupConversationID(ctx, host, "s"))

	// ...and the next write overwrites it with a valid, readable store.
	storeConversationID(ctx, host, "s", "conv-9")
	assert.Equal(t, "conv-9", lookupConversationID(ctx, host, "s"))
}
