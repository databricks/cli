package terraform

import "strings"

// TODO: This is temporary. Come up with a robust way to log mutator progress and
// status events
// TODO: consolidate json cmdio logs with tty detection
type ResourceChange struct {
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
	ResourceName string `json:"resource_name"`
}

func (c *ResourceChange) String() string {
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
