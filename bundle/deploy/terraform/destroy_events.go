package terraform

import "strings"

type PlanResourceChange struct {
	Type         string `json:"type"`
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
	ResourceName string `json:"resource_name"`
}

func (c *PlanResourceChange) String() string {
	result := strings.Builder{}
	switch c.Action {
	case "delete":
		result.WriteString("  delete ")
	default:
		result.WriteString(c.Action + " ")
	}
	switch c.ResourceType {
	case "databricks_job":
		result.WriteString("job ")
	case "databricks_pipeline":
		result.WriteString("pipeline ")
	default:
		result.WriteString(c.ResourceType + " ")
	}
	result.WriteString(c.ResourceName)
	return result.String()
}

type DestroyEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (event *DestroyEvent) String() string {
	return event.Message
}

func NewDestroyStartEvent() *DestroyEvent {
	return &DestroyEvent{
		Type:    "terraform_destroy_started_event",
		Message: "Starting to destroy resources",
	}
}

func NewDestroyCompletedEvent() *DestroyEvent {
	return &DestroyEvent{
		Type:    "terraform_destroy_completed_event",
		Message: "Successfully destroyed resources!",
	}
}

func NewDestroyFailedEvent() *DestroyEvent {
	return &DestroyEvent{
		Type:    "terraform_destroy_failed_event",
		Message: "Failed to destroy resources",
	}
}

func NewDestroySkippedEvent() *DestroyEvent {
	return &DestroyEvent{
		Type:    "terraform_destroy_skipped_event",
		Message: "No resources to destroy in plan. Skipping destroy!",
	}
}

func NewDestroyPlanWarningEvent() *DestroyEvent {
	return &DestroyEvent{
		Type:    "terraform_destroy_plan_warning_event",
		Message: "The following resources will be removed:",
	}
}
