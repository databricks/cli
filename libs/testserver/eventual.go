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

// EventualMap is a keyed collection of EventualValues that share one
// eventual-consistency mode. When eventualConsistency is enabled, the first Read
// of a key after a Write returns the pre-write value (the zero value for a
// freshly created key, e.g. a 404 for a not-yet-propagated resource).
//
// Not safe for concurrent use; callers must hold the workspace lock.
type EventualMap[K comparable, V any] struct {
	values              map[K]*EventualValue[V]
	eventualConsistency bool
}

func NewEventualMap[K comparable, V any](eventualConsistency bool) *EventualMap[K, V] {
	return &EventualMap[K, V]{
		values:              map[K]*EventualValue[V]{},
		eventualConsistency: eventualConsistency,
	}
}

// Read returns the value for key, applying eventual consistency when enabled. ok
// reports whether the key exists; value may be the zero value when a write to an
// existing key has not propagated yet.
func (m *EventualMap[K, V]) Read(key K) (value V, ok bool) {
	ev, ok := m.values[key]
	if !ok {
		return value, false
	}
	if m.eventualConsistency {
		return ev.ReadEventual(), true
	}
	return ev.ReadStrong(), true
}

// ReadStrong returns the current value for key without consuming stale state.
// Use it from write handlers that need the latest committed value.
func (m *EventualMap[K, V]) ReadStrong(key K) (value V, ok bool) {
	ev, ok := m.values[key]
	if !ok {
		return value, false
	}
	return ev.ReadStrong(), true
}

// Write stages v for key; the next Read returns the pre-write value.
func (m *EventualMap[K, V]) Write(key K, v V) {
	ev := m.values[key]
	if ev == nil {
		ev = &EventualValue[V]{}
		m.values[key] = ev
	}
	ev.Write(v)
}

// Put sets v for key immediately, discarding any pending stale state.
func (m *EventualMap[K, V]) Put(key K, v V) {
	ev := m.values[key]
	if ev == nil {
		ev = &EventualValue[V]{}
		m.values[key] = ev
	}
	ev.Put(v)
}
