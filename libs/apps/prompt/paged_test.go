package prompt

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagedFetcherIsDoneNilChannel(t *testing.T) {
	f := &PagedFetcher{}
	assert.True(t, f.IsDone())
}

func TestPagedFetcherIsDoneBeforeClose(t *testing.T) {
	f := &PagedFetcher{done: make(chan struct{})}
	assert.False(t, f.IsDone())
}

func TestPagedFetcherIsDoneAfterClose(t *testing.T) {
	f := &PagedFetcher{done: make(chan struct{})}
	close(f.done)
	assert.True(t, f.IsDone())
}

func TestPagedFetcherWaitForFirstPageNilChannel(t *testing.T) {
	f := &PagedFetcher{Err: errors.New("failed")}
	err := f.WaitForFirstPage(t.Context())
	assert.EqualError(t, err, "failed")
}

func TestPagedFetcherWaitForFirstPageSuccess(t *testing.T) {
	f := &PagedFetcher{done: make(chan struct{})}
	go func() {
		f.Items = []ListItem{{ID: "1"}}
		close(f.done)
	}()
	err := f.WaitForFirstPage(t.Context())
	assert.NoError(t, err)
	assert.Len(t, f.Items, 1)
}

func TestPagedFetcherWaitForFirstPageContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	f := &PagedFetcher{done: make(chan struct{})}
	cancel()

	err := f.WaitForFirstPage(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPagedFetcherWaitForFirstPageError(t *testing.T) {
	f := &PagedFetcher{done: make(chan struct{}), Err: errors.New("api error")}
	close(f.done)
	err := f.WaitForFirstPage(t.Context())
	assert.EqualError(t, err, "api error")
}

func TestPagedFetcherLoadMoreAppendsItems(t *testing.T) {
	f := &PagedFetcher{
		Items:   []ListItem{{ID: "1"}},
		HasMore: true,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return []ListItem{{ID: "2"}, {ID: "3"}}, false, nil
		},
	}

	err := f.LoadMore(t.Context())
	require.NoError(t, err)
	assert.Len(t, f.Items, 3)
	assert.False(t, f.HasMore)
}

func TestPagedFetcherLoadMoreCapsAtLimit(t *testing.T) {
	items := make([]ListItem, maxTotalResults-1)
	for i := range items {
		items[i] = ListItem{ID: "x"}
	}
	f := &PagedFetcher{
		Items:   items,
		HasMore: true,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return []ListItem{{ID: "last"}, {ID: "overflow"}}, true, nil
		},
	}

	err := f.LoadMore(t.Context())
	require.NoError(t, err)
	assert.False(t, f.HasMore)
	assert.True(t, f.Capped)
}

func TestPagedFetcherLoadMoreNoopWhenNotHasMore(t *testing.T) {
	f := &PagedFetcher{
		Items:   []ListItem{{ID: "1"}},
		HasMore: false,
	}
	err := f.LoadMore(t.Context())
	assert.NoError(t, err)
	assert.Len(t, f.Items, 1)
}

func TestPagedFetcherLoadMoreNoopWhenNilFunc(t *testing.T) {
	f := &PagedFetcher{HasMore: true, loadMore: nil}
	err := f.LoadMore(t.Context())
	assert.NoError(t, err)
}

func TestPagedFetcherLoadMoreError(t *testing.T) {
	f := &PagedFetcher{
		Items:   []ListItem{{ID: "1"}},
		HasMore: true,
		loadMore: func(ctx context.Context) ([]ListItem, bool, error) {
			return nil, false, errors.New("network error")
		},
	}

	err := f.LoadMore(t.Context())
	assert.EqualError(t, err, "network error")
	assert.Len(t, f.Items, 1)
}

// stubIterator implements listing.Iterator for testing collectN.
type stubIterator struct {
	items []string
	pos   int
}

func (s *stubIterator) HasNext(_ context.Context) bool {
	return s.pos < len(s.items)
}

func (s *stubIterator) Next(_ context.Context) (string, error) {
	if s.pos >= len(s.items) {
		return "", errors.New("exhausted")
	}
	item := s.items[s.pos]
	s.pos++
	return item, nil
}

func TestCollectNFullPage(t *testing.T) {
	iter := &stubIterator{items: []string{"a", "b", "c", "d", "e"}}
	items, hasMore, err := collectN(t.Context(), iter, 3, func(s string) ListItem {
		return ListItem{ID: s, Label: s}
	})

	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.True(t, hasMore)
	assert.Equal(t, "a", items[0].ID)
	assert.Equal(t, "c", items[2].ID)
}

func TestCollectNExhaustsIterator(t *testing.T) {
	iter := &stubIterator{items: []string{"a", "b"}}
	items, hasMore, err := collectN(t.Context(), iter, 5, func(s string) ListItem {
		return ListItem{ID: s, Label: s}
	})

	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.False(t, hasMore)
}

func TestCollectNEmptyIterator(t *testing.T) {
	iter := &stubIterator{items: nil}
	items, hasMore, err := collectN(t.Context(), iter, 5, func(s string) ListItem {
		return ListItem{ID: s, Label: s}
	})

	require.NoError(t, err)
	assert.Empty(t, items)
	assert.False(t, hasMore)
}
