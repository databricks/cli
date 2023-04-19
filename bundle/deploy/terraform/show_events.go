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
	Action       Action `json:"action"`
}

type Action string

const (
	ActionCreate  = Action("create")
	ActionUpdate  = Action("update")
	ActionDelete  = Action("delete")
	ActionReplace = Action("replace")
	ActionNoop    = Action("no-op")
)

func toAction(actions tfjson.Actions) Action {
	action := ActionNoop
	switch {
	case actions.Create():
		action = ActionCreate
	case actions.Update():
		action = ActionUpdate
	case actions.Delete():
		action = ActionDelete
	case actions.Replace():
		action = ActionReplace
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
	action := string(event.Action)

	// color create, replace and delete events
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

	return strings.Join([]string{" ", action, event.ResourceType, event.Name}, " ")
}

func (event *ResourceChangeEvent) IsInplaceSupported() bool {
	return false
}
