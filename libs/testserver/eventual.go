package testserver

// EventualValue provides eventual-consistency read semantics for a single
// value. After Write, the first Read returns the pre-write value; subsequent
// Reads return the written value.
//
// Not safe for concurrent use; callers must hold the workspace lock.
type EventualValue[T any] struct {
	current T
	stale   T
	pending bool
}

// Write stages v as the new value. The pre-write value is returned by the
// next ReadEventual call.
func (e *EventualValue[T]) Write(v T) {
	e.stale = e.current
	e.current = v
	e.pending = true
}

// ReadEventual returns the current value, applying eventual-consistency: the
// first call after a Write returns the pre-write (stale) value.
func (e *EventualValue[T]) ReadEventual() T {
	if e.pending {
		e.pending = false
		return e.stale
	}
	return e.current
}

// ReadStrong returns the current value without consuming the pending stale
// state. Use for write operations that need the latest committed value.
func (e *EventualValue[T]) ReadStrong() T {
	return e.current
}

// Put sets v as the current value immediately without staging a stale
// version. Any pending stale state is discarded.
func (e *EventualValue[T]) Put(v T) {
	e.current = v
	e.pending = false
}
