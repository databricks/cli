// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlAlertOptions struct {
	Column           string `json:"column"`
	CustomBody       string `json:"custom_body,omitempty"`
	CustomSubject    string `json:"custom_subject,omitempty"`
	EmptyResultState string `json:"empty_result_state,omitempty"`
	Muted            bool   `json:"muted,omitempty"`
	Op               string `json:"op"`
	Value            string `json:"value"`
}

type ResourceSqlAlertProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceSqlAlert struct {
	CreatedAt      string                          `json:"created_at,omitempty"`
	Id             string                          `json:"id,omitempty"`
	Name           string                          `json:"name"`
	Parent         string                          `json:"parent,omitempty"`
	QueryId        string                          `json:"query_id"`
	Rearm          int                             `json:"rearm,omitempty"`
	UpdatedAt      string                          `json:"updated_at,omitempty"`
	Options        *ResourceSqlAlertOptions        `json:"options,omitempty"`
	ProviderConfig *ResourceSqlAlertProviderConfig `json:"provider_config,omitempty"`
}
