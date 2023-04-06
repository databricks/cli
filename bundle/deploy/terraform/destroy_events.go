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
