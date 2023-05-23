// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceClusterPolicy struct {
	Definition                      string `json:"definition,omitempty"`
	Description                     string `json:"description,omitempty"`
	Id                              string `json:"id,omitempty"`
	MaxClustersPerUser              int    `json:"max_clusters_per_user,omitempty"`
	Name                            string `json:"name"`
	PolicyFamilyDefinitionOverrides string `json:"policy_family_definition_overrides,omitempty"`
	PolicyFamilyId                  string `json:"policy_family_id,omitempty"`
	PolicyId                        string `json:"policy_id,omitempty"`
}
