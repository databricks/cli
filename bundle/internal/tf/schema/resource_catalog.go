// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceCatalog struct {
	Comment                      string            `json:"comment,omitempty"`
	ConnectionName               string            `json:"connection_name,omitempty"`
	EnablePredictiveOptimization string            `json:"enable_predictive_optimization,omitempty"`
	ForceDestroy                 bool              `json:"force_destroy,omitempty"`
	Id                           string            `json:"id,omitempty"`
	IsolationMode                string            `json:"isolation_mode,omitempty"`
	MetastoreId                  string            `json:"metastore_id,omitempty"`
	Name                         string            `json:"name"`
	Options                      map[string]string `json:"options,omitempty"`
	Owner                        string            `json:"owner,omitempty"`
	Properties                   map[string]string `json:"properties,omitempty"`
	ProviderName                 string            `json:"provider_name,omitempty"`
	ShareName                    string            `json:"share_name,omitempty"`
	StorageRoot                  string            `json:"storage_root,omitempty"`
}
