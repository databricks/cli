package statemgmt

// ResourcesJSON is the DMS-managed resources.json format.
// When DMS is enabled, resources.json stores only the deployment ID.
type ResourcesJSON struct {
	DeploymentID string `json:"deployment_id"`
}
