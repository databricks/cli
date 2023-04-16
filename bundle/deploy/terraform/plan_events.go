package terraform

import "fmt"

type PlanEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (event *PlanEvent) String() string {
	return event.Message
}

func NewPlanningStartedEvent() *PlanEvent {
	return &PlanEvent{
		Type:    "terraform_plan_started_event",
		Message: "Starting plan computation",
	}
}

func NewPlanningFailedEvent() *PlanEvent {
	return &PlanEvent{
		Type:    "terraform_plan_failed_event",
		Message: "Failed to compute plan",
	}
}

func NewPlanningCompletedEvent(planPath string) *PlanEvent {
	return &PlanEvent{
		Type:    "terraform_plan_completed_event",
		Message: fmt.Sprintf("Planning complete and persisted at %s\n", planPath),
	}
}
