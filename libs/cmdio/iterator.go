package cmdio

import (
	"context"
	"iter"
)

// pullIterator adapts an iter.Seq2 (what sdk-go list methods return) to the
// HasNext/Next style the renderers use to peek before writing separators.
type pullIterator[T any] struct {
	next   func() (T, error, bool)
	stop   func()
	peeked bool
	done   bool
	value  T
	err    error
}

func newPullIterator[T any](seq iter.Seq2[T, error]) *pullIterator[T] {
	next, stop := iter.Pull2(seq)
	return &pullIterator[T]{next: next, stop: stop}
}

func (p *pullIterator[T]) HasNext(_ context.Context) bool {
	if !p.peeked && !p.done {
		var ok bool
		p.value, p.err, ok = p.next()
		p.peeked = true
		if !ok {
			p.done = true
			p.stop()
		}
	}
	return !p.done
}

func (p *pullIterator[T]) Next(ctx context.Context) (T, error) {
	p.HasNext(ctx)
	p.peeked = false
	return p.value, p.err
}

// Close releases the underlying sequence if it was not fully consumed.
func (p *pullIterator[T]) Close() {
	p.stop()
}

type pullIter[T any] interface {
	HasNext(context.Context) bool
	Next(context.Context) (T, error)
}
