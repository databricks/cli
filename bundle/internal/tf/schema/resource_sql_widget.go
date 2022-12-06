// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlWidgetParameter struct {
	MapTo  string   `json:"map_to,omitempty"`
	Name   string   `json:"name"`
	Title  string   `json:"title,omitempty"`
	Type   string   `json:"type"`
	Value  string   `json:"value,omitempty"`
	Values []string `json:"values,omitempty"`
}

type ResourceSqlWidgetPosition struct {
	AutoHeight bool `json:"auto_height,omitempty"`
	PosX       int  `json:"pos_x,omitempty"`
	PosY       int  `json:"pos_y,omitempty"`
	SizeX      int  `json:"size_x"`
	SizeY      int  `json:"size_y"`
}

type ResourceSqlWidget struct {
	DashboardId     string                       `json:"dashboard_id"`
	Description     string                       `json:"description,omitempty"`
	Id              string                       `json:"id,omitempty"`
	Text            string                       `json:"text,omitempty"`
	Title           string                       `json:"title,omitempty"`
	VisualizationId string                       `json:"visualization_id,omitempty"`
	WidgetId        string                       `json:"widget_id,omitempty"`
	Parameter       []ResourceSqlWidgetParameter `json:"parameter,omitempty"`
	Position        *ResourceSqlWidgetPosition   `json:"position,omitempty"`
}
