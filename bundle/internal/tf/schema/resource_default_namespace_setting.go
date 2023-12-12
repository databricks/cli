// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDefaultNamespaceSettingNamespace struct {
	Value string `json:"value,omitempty"`
}

type ResourceDefaultNamespaceSetting struct {
	Etag        string                                    `json:"etag,omitempty"`
	Id          string                                    `json:"id,omitempty"`
	SettingName string                                    `json:"setting_name,omitempty"`
	Namespace   *ResourceDefaultNamespaceSettingNamespace `json:"namespace,omitempty"`
}
