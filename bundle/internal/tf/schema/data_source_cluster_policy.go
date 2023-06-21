// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceClusterPolicy struct {
	Definition                      string `json:"definition,omitempty"`
	Description                     string `json:"description,omitempty"`
	Id                              string `json:"id,omitempty"`
	IsDefault                       bool   `json:"is_default,omitempty"`
	MaxClustersPerUser              int    `json:"max_clusters_per_user,omitempty"`
	Name                            string `json:"name,omitempty"`
	PolicyFamilyDefinitionOverrides string `json:"policy_family_definition_overrides,omitempty"`
	PolicyFamilyId                  string `json:"policy_family_id,omitempty"`
}
