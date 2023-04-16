package terraform

type ApplyEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (event *ApplyEvent) String() string {
	return event.Message
}

func NewApplyStartedEvent() *ApplyEvent {
	return &ApplyEvent{
		Type:    "terraform_apply_started_event",
		Message: "Starting resource deployment",
	}
}

func NewApplyFailedEvent() *ApplyEvent {
	return &ApplyEvent{
		Type:    "terraform_apply_failed_event",
		Message: "Failed to deploy resources",
	}
}

func NewApplyCompletedEvent() *ApplyEvent {
	return &ApplyEvent{
		Type:    "terraform_apply_completed_event",
		Message: "Resource deployment completed!",
	}
}
