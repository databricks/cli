package listing

import (
	"context"
	"iter"
)

// SeqIterator adapts an iter.Seq2, which is what sdk-go list methods return,
// to the pull-style Iterator.
type SeqIterator[T any] struct {
	next   func() (T, error, bool)
	stop   func()
	peeked bool
	done   bool
	value  T
	err    error
}

// FromSeq returns an Iterator over seq. Call Close if the iterator may not be
// consumed to the end, to release the sequence.
func FromSeq[T any](seq iter.Seq2[T, error]) *SeqIterator[T] {
	next, stop := iter.Pull2(seq)
	return &SeqIterator[T]{next: next, stop: stop}
}

func (s *SeqIterator[T]) HasNext(_ context.Context) bool {
	if !s.peeked && !s.done {
		var ok bool
		s.value, s.err, ok = s.next()
		s.peeked = true
		if !ok {
			s.done = true
			s.stop()
		}
	}
	return !s.done
}

func (s *SeqIterator[T]) Next(ctx context.Context) (T, error) {
	if !s.HasNext(ctx) {
		var zero T
		return zero, ErrNoMoreItems
	}
	s.peeked = false
	return s.value, s.err
}

// Close releases the underlying sequence.
func (s *SeqIterator[T]) Close() {
	s.stop()
}
