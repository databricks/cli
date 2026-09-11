// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDisasterRecoveryStableUrl struct {
	EffectiveWorkspaceId string `json:"effective_workspace_id,omitempty"`
	FailoverGroupName    string `json:"failover_group_name,omitempty"`
	InitialWorkspaceId   string `json:"initial_workspace_id"`
	Name                 string `json:"name,omitempty"`
	Parent               string `json:"parent"`
	StableUrlId          string `json:"stable_url_id"`
	StableWorkspaceId    string `json:"stable_workspace_id,omitempty"`
	Url                  string `json:"url,omitempty"`
}
