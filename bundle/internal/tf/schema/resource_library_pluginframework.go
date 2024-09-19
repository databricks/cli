// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceLibraryPluginframework struct {
	ClusterId    string `json:"cluster_id"`
	Egg          string `json:"egg,omitempty"`
	Jar          string `json:"jar,omitempty"`
	Requirements string `json:"requirements,omitempty"`
	Whl          string `json:"whl,omitempty"`
}
