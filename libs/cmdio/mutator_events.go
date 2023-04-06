package cmdio

import "context"

type MutatorEventType string

const (
	MutatorRunning   = MutatorEventType("running")
	MutatorCompleted = MutatorEventType("completed")
	MutatorFailed    = MutatorEventType("failed")
)

type MutatorEvent struct {
	Type    string           `json:"type"`
	Status  MutatorEventType `json:"status"`
	Source  string           `json:"source"`
	Message string           `json:"message"`
}

func (event *MutatorEvent) String() string {
	return event.Message
}

func LogMutatorEvent(ctx context.Context, name string, eventType MutatorEventType, message string) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(&MutatorEvent{
		Type:    "mutator_event",
		Status:  eventType,
		Source:  name,
		Message: message,
	})
}
