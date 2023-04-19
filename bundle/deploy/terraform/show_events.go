package terraform

import (
	"os"
	"strings"

	"github.com/fatih/color"
	tfjson "github.com/hashicorp/terraform-json"
	"golang.org/x/term"
)

type ResourceChangeEvent struct {
	Name         string `json:"name"`
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
}

func toAction(actions tfjson.Actions) string {
	action := "no-op"
	switch {
	case actions.Create():
		action = "create"
	case actions.Read():
		action = "read"
	case actions.Update():
		action = "update"
	case actions.Delete():
		action = "delete"
	case actions.Replace():
		action = "replace"
	}

	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	isTty := term.IsTerminal(int(os.Stderr.Fd()))
	if isTty && action == "create" {
		action = green(action)
	}
	if isTty && action == "delete" {
		action = red(action)
	}
	if isTty && action == "replace" {
		action = yellow(action)
	}
	return action
}

func toResourceType(terraformType string) string {
	switch terraformType {
	case "databricks_job":
		return "job"
	case "databricks_pipeline":
		return "pipeline"
	case "databricks_mlflow_model":
		return "mlflow_model"
	case "databricks_mlflow_experiment":
		return "mlflow_experiment"
	default:
		return ""
	}
}

func toResourceChangeEvent(change *tfjson.ResourceChange) *ResourceChangeEvent {
	if change.Change == nil {
		return nil
	}
	actions := change.Change.Actions
	if actions.Read() || actions.NoOp() {
		return nil
	}
	action := toAction(actions)

	resourceType := toResourceType(change.Type)
	if resourceType == "" {
		return nil
	}

	name := change.Name
	if name == "" {
		return nil
	}

	return &ResourceChangeEvent{
		Name:         name,
		Action:       action,
		ResourceType: resourceType,
	}
}

func (event *ResourceChangeEvent) String() string {
	return strings.Join([]string{" ", string(event.Action), event.ResourceType, event.Name}, " ")
}

func (event *ResourceChangeEvent) IsInplaceSupported() bool {
	return false
}
