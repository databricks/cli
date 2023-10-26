// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlAlertOptions struct {
	Column        string `json:"column"`
	CustomBody    string `json:"custom_body,omitempty"`
	CustomSubject string `json:"custom_subject,omitempty"`
	Muted         bool   `json:"muted,omitempty"`
	Op            string `json:"op"`
	Value         string `json:"value"`
}

type ResourceSqlAlert struct {
	Id      string                   `json:"id,omitempty"`
	Name    string                   `json:"name"`
	Parent  string                   `json:"parent,omitempty"`
	QueryId string                   `json:"query_id"`
	Rearm   int                      `json:"rearm,omitempty"`
	Options *ResourceSqlAlertOptions `json:"options,omitempty"`
}
