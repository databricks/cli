package geniecmd

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const askTestHost = "https://example.cloud.databricks.com"

// fakeAsk returns queued results in order and records the server ids it saw.
type fakeAsk struct {
	results []askResult
	calls   []string
}

type askResult struct {
	id    string
	wrote bool
	err   error
}

func (f *fakeAsk) run(serverID string) (string, bool, error) {
	f.calls = append(f.calls, serverID)
	r := f.results[len(f.calls)-1]
	return r.id, r.wrote, r.err
}

func TestAskWithConversation(t *testing.T) {
	boom := errors.New("boom")

	t.Run("first use stores the new conversation", func(t *testing.T) {
		ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
		f := &fakeAsk{results: []askResult{{id: "new", wrote: true}}}
		require.NoError(t, askWithConversation(ctx, &bytes.Buffer{}, askTestHost, "q", f.run))
		assert.Equal(t, []string{""}, f.calls)
		assert.Equal(t, "new", lookupConversationID(ctx, askTestHost, "q"))
	})

	t.Run("reuse sends the stored id and refreshes it", func(t *testing.T) {
		ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
		storeConversationID(ctx, askTestHost, "q", "srv")
		f := &fakeAsk{results: []askResult{{id: "srv", wrote: true}}}
		require.NoError(t, askWithConversation(ctx, &bytes.Buffer{}, askTestHost, "q", f.run))
		assert.Equal(t, []string{"srv"}, f.calls)
		assert.Equal(t, "srv", lookupConversationID(ctx, askTestHost, "q"))
	})

	t.Run("dead mapping fails open: forget, note, retry fresh, remap", func(t *testing.T) {
		ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
		storeConversationID(ctx, askTestHost, "q", "dead")
		var stderr bytes.Buffer
		f := &fakeAsk{results: []askResult{{err: boom}, {id: "fresh", wrote: true}}}
		require.NoError(t, askWithConversation(ctx, &stderr, askTestHost, "q", f.run))
		assert.Equal(t, []string{"dead", ""}, f.calls)
		assert.Contains(t, stderr.String(), "was not found")
		assert.Equal(t, "fresh", lookupConversationID(ctx, askTestHost, "q"))
	})

	t.Run("error after output is not retried; mapping forgotten", func(t *testing.T) {
		ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
		storeConversationID(ctx, askTestHost, "q", "dead")
		f := &fakeAsk{results: []askResult{{wrote: true, err: boom}}}
		err := askWithConversation(ctx, &bytes.Buffer{}, askTestHost, "q", f.run)
		assert.ErrorIs(t, err, boom)
		assert.Equal(t, []string{"dead"}, f.calls)
		assert.Empty(t, lookupConversationID(ctx, askTestHost, "q"))
	})

	t.Run("cancel keeps the mapping and does not retry", func(t *testing.T) {
		ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
		storeConversationID(ctx, askTestHost, "q", "live")
		f := &fakeAsk{results: []askResult{{err: context.Canceled}}}
		err := askWithConversation(ctx, &bytes.Buffer{}, askTestHost, "q", f.run)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, []string{"live"}, f.calls)
		assert.Equal(t, "live", lookupConversationID(ctx, askTestHost, "q"))
	})

	t.Run("empty label runs once and stores nothing", func(t *testing.T) {
		ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
		f := &fakeAsk{results: []askResult{{id: "x", wrote: true}}}
		require.NoError(t, askWithConversation(ctx, &bytes.Buffer{}, askTestHost, "", f.run))
		assert.Equal(t, []string{""}, f.calls)
	})
}
