package logstream

import (
	"context"
	"fmt"
	"io"
	"time"
)

// consumeState manages the tail buffer and flush timing during log consumption.
type consumeState struct {
	buffer        *tailBuffer
	writer        io.Writer
	flushed       bool
	flushDeadline time.Time
	tail          int
	follow        bool
}

func newConsumeState(tail int, follow bool, prefetch time.Duration, writer io.Writer, alreadyFlushed bool) *consumeState {
	s := &consumeState{
		buffer:  &tailBuffer{size: tail},
		writer:  writer,
		flushed: tail == 0 || alreadyFlushed,
		tail:    tail,
		follow:  follow,
	}
	if tail > 0 && prefetch > 0 && !alreadyFlushed {
		s.flushDeadline = time.Now().Add(prefetch)
	}
	return s
}

// ReadDeadline returns the effective read deadline considering context and flush deadline.
func (s *consumeState) ReadDeadline(ctx context.Context) time.Time {
	ctxDeadline, hasCtxDeadline := ctx.Deadline()

	if !s.flushDeadline.IsZero() {
		if !hasCtxDeadline || s.flushDeadline.Before(ctxDeadline) {
			return s.flushDeadline
		}
	}

	if hasCtxDeadline {
		return ctxDeadline
	}
	return time.Time{}
}

// HasPendingFlushDeadline returns true if a prefetch flush deadline is pending.
func (s *consumeState) HasPendingFlushDeadline() bool {
	return !s.flushDeadline.IsZero()
}

// HandleFlushTimeout handles a prefetch timeout by flushing the buffer.
// Only call this when HasPendingFlushDeadline() returns true.
// Returns true if reading should continue (following), false if done (not following).
func (s *consumeState) HandleFlushTimeout() (shouldContinue bool, err error) {
	s.flushDeadline = time.Time{}
	if s.tail > 0 && !s.flushed {
		if err := s.buffer.Flush(s.writer); err != nil {
			return false, err
		}
		s.flushed = true
	}
	return s.follow, nil
}

// ProcessLine either buffers or writes the line depending on flush state.
func (s *consumeState) ProcessLine(line string) error {
	if s.tail > 0 && !s.flushed {
		s.buffer.Add(line)
		if s.flushDeadline.IsZero() && s.buffer.Len() >= s.tail && s.follow {
			if err := s.buffer.Flush(s.writer); err != nil {
				return err
			}
			s.flushed = true
		}
		return nil
	}
	_, err := fmt.Fprintln(s.writer, line)
	return err
}

// FlushRemaining flushes any remaining buffered lines.
func (s *consumeState) FlushRemaining() error {
	if s.tail > 0 && !s.flushed {
		if err := s.buffer.Flush(s.writer); err != nil {
			return err
		}
		s.flushed = true
	}
	return nil
}

// IsFlushed returns whether the buffer has been flushed.
func (s *consumeState) IsFlushed() bool {
	return s.flushed
}
