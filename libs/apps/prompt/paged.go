package prompt

import (
	"context"

	"github.com/databricks/databricks-sdk-go/listing"
)

const (
	pageSize        = 50
	maxTotalResults = 500
	moreID          = "__more__"
	manualID        = "__manual__"
)

// PagedFetcher provides incremental access to a resource list. The first page
// is loaded in a background goroutine (signaled via the done channel).
// Subsequent pages are loaded on demand via LoadMore. Once maxTotalResults
// items have been accumulated, Capped is set to true and no more pages are
// offered — only the manual input fallback.
type PagedFetcher struct {
	Items   []ListItem
	HasMore bool
	Capped  bool
	Err     error

	done     chan struct{} // closed when the first page is ready
	loadMore func(ctx context.Context) ([]ListItem, bool, error)
}

// WaitForFirstPage blocks until the first page is ready or the context is cancelled.
func (p *PagedFetcher) WaitForFirstPage(ctx context.Context) error {
	if p.done == nil {
		return p.Err
	}
	select {
	case <-p.done:
		return p.Err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsDone returns true if the first page has already been loaded.
func (p *PagedFetcher) IsDone() bool {
	if p.done == nil {
		return true
	}
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// LoadMore fetches the next page and appends it to Items. If the total reaches
// maxTotalResults, HasMore is cleared and Capped is set.
func (p *PagedFetcher) LoadMore(ctx context.Context) error {
	if !p.HasMore || p.loadMore == nil {
		return nil
	}
	items, hasMore, err := p.loadMore(ctx)
	if err != nil {
		return err
	}
	p.Items = append(p.Items, items...)
	p.HasMore = hasMore
	if len(p.Items) >= maxTotalResults {
		p.HasMore = false
		p.Capped = true
	}
	return nil
}

// collectN consumes up to n items from an SDK iterator, mapping each to a
// ListItem. Returns the items, whether more exist, and any error.
func collectN[T any](ctx context.Context, iter listing.Iterator[T], n int, mapFn func(T) ListItem) ([]ListItem, bool, error) {
	var items []ListItem
	for len(items) < n {
		if !iter.HasNext(ctx) {
			return items, false, nil
		}
		item, err := iter.Next(ctx)
		if err != nil {
			return items, false, err
		}
		items = append(items, mapFn(item))
	}
	return items, iter.HasNext(ctx), nil
}
