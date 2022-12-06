// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlVisualization struct {
	Description     string `json:"description,omitempty"`
	Id              string `json:"id,omitempty"`
	Name            string `json:"name"`
	Options         string `json:"options"`
	QueryId         string `json:"query_id"`
	QueryPlan       string `json:"query_plan,omitempty"`
	Type            string `json:"type"`
	VisualizationId string `json:"visualization_id,omitempty"`
}
