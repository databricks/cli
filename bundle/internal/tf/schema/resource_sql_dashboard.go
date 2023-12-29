// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlDashboard struct {
	CreatedAt               string   `json:"created_at,omitempty"`
	DashboardFiltersEnabled bool     `json:"dashboard_filters_enabled,omitempty"`
	Id                      string   `json:"id,omitempty"`
	Name                    string   `json:"name"`
	Parent                  string   `json:"parent,omitempty"`
	RunAsRole               string   `json:"run_as_role,omitempty"`
	Tags                    []string `json:"tags,omitempty"`
	UpdatedAt               string   `json:"updated_at,omitempty"`
}
