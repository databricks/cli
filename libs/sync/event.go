package sync

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type EventType string

const (
	EventTypeStart    = EventType("start")
	EventTypeProgress = EventType("progress")
	EventTypeComplete = EventType("complete")
)

type EventAction string

const (
	EventActionPut    = EventAction("put")
	EventActionDelete = EventAction("delete")
)

type Event interface {
	fmt.Stringer
}

type EventBase struct {
	Timestamp time.Time `json:"timestamp"`
	Seq       int       `json:"seq"`
	Type      EventType `json:"type"`
}

func newEventBase(seq int, typ EventType) *EventBase {
	return &EventBase{
		Timestamp: time.Now(),
		Seq:       seq,
		Type:      typ,
	}
}

type EventChanges struct {
	Put    []string `json:"put,omitempty"`
	Delete []string `json:"delete,omitempty"`
}

func (e *EventChanges) IsEmpty() bool {
	return len(e.Put) == 0 && len(e.Delete) == 0
}

func (e *EventChanges) String() string {
	var changes []string
	if len(e.Put) > 0 {
		changes = append(changes, fmt.Sprintf("PUT: %s", strings.Join(e.Put, ", ")))
	}
	if len(e.Delete) > 0 {
		changes = append(changes, fmt.Sprintf("DELETE: %s", strings.Join(e.Delete, ", ")))
	}
	return strings.Join(changes, ", ")
}

type EventStart struct {
	*EventBase
	*EventChanges
}

func (e *EventStart) String() string {
	if e.IsEmpty() {
		return ""
	}

	return fmt.Sprintf("Action: %s", e.EventChanges.String())
}

func newEventStart(seq int, put []string, delete []string) Event {
	return &EventStart{
		EventBase:    newEventBase(seq, EventTypeStart),
		EventChanges: &EventChanges{Put: put, Delete: delete},
	}
}

type EventSyncProgress struct {
	*EventBase

	Action EventAction `json:"action"`
	Path   string      `json:"path"`

	// Progress is in range [0, 1] where 0 means the operation started
	// and 1 means the operation completed.
	Progress float32 `json:"progress"`
}

func (e *EventSyncProgress) String() string {
	if e.Progress < 1.0 {
		return ""
	}

	switch e.Action {
	case EventActionPut:
		return fmt.Sprintf("Uploaded %s", e.Path)
	case EventActionDelete:
		return fmt.Sprintf("Deleted %s", e.Path)
	default:
		panic("invalid action")
	}
}

func newEventProgress(seq int, action EventAction, path string, progress float32) Event {
	return &EventSyncProgress{
		EventBase: newEventBase(seq, EventTypeProgress),

		Action:   action,
		Path:     path,
		Progress: progress,
	}
}

type EventSyncComplete struct {
	*EventBase
	*EventChanges
}

func (e *EventSyncComplete) String() string {
	if e.Seq == 0 {
		return "Initial Sync Complete"
	}

	if e.IsEmpty() {
		return ""
	}

	return "Complete"
}

func newEventComplete(seq int, put []string, delete []string) Event {
	return &EventSyncComplete{
		EventBase:    newEventBase(seq, EventTypeComplete),
		EventChanges: &EventChanges{Put: put, Delete: delete},
	}
}

type EventNotifier interface {
	Notify(ctx context.Context, event Event)
}

// ChannelNotifier implements [EventNotifier] and sends events to its channel.
type ChannelNotifier struct {
	ch chan<- Event
}

func (n *ChannelNotifier) Notify(ctx context.Context, e Event) {
	select {
	case <-ctx.Done():
	case n.ch <- e:
	}
}

// NopNotifier implements [EventNotifier] and does nothing.
type NopNotifier struct{}

func (n *NopNotifier) Notify(ctx context.Context, e Event) {
	// Discard
}
