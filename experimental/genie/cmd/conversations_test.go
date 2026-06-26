package geniecmd

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestConversationCache(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_CACHE_DIR", t.TempDir())
	c := newConversationCache(ctx)
	const host = "https://example.cloud.databricks.com"

	// Miss before anything is stored, and an empty label never maps.
	assert.Empty(t, lookupConversationID(ctx, c, host, "q3"))
	assert.Empty(t, lookupConversationID(ctx, c, host, ""))

	// Store then resolve.
	storeConversationID(ctx, c, host, "q3", "server-1")
	assert.Equal(t, "server-1", lookupConversationID(ctx, c, host, "q3"))

	// Labels are scoped by host, and distinct labels are independent.
	assert.Empty(t, lookupConversationID(ctx, c, "https://other.databricks.com", "q3"))
	assert.Empty(t, lookupConversationID(ctx, c, host, "q4"))

	// Forgetting a stale mapping makes the label resolve fresh again.
	forgetConversation(ctx, c, host, "q3")
	assert.Empty(t, lookupConversationID(ctx, c, host, "q3"))
}
