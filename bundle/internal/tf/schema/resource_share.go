// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceShareObjectPartitionValue struct {
	Name                 string `json:"name"`
	Op                   string `json:"op"`
	RecipientPropertyKey string `json:"recipient_property_key,omitempty"`
	Value                string `json:"value,omitempty"`
}

type ResourceShareObjectPartition struct {
	Value []ResourceShareObjectPartitionValue `json:"value,omitempty"`
}

type ResourceShareObject struct {
	AddedAt                           int                            `json:"added_at,omitempty"`
	AddedBy                           string                         `json:"added_by,omitempty"`
	CdfEnabled                        bool                           `json:"cdf_enabled,omitempty"`
	Comment                           string                         `json:"comment,omitempty"`
	Content                           string                         `json:"content,omitempty"`
	DataObjectType                    string                         `json:"data_object_type"`
	EffectiveCdfEnabled               bool                           `json:"effective_cdf_enabled,omitempty"`
	EffectiveHistoryDataSharingStatus string                         `json:"effective_history_data_sharing_status,omitempty"`
	EffectiveSharedAs                 string                         `json:"effective_shared_as,omitempty"`
	EffectiveStartVersion             int                            `json:"effective_start_version,omitempty"`
	EffectiveStringSharedAs           string                         `json:"effective_string_shared_as,omitempty"`
	HistoryDataSharingStatus          string                         `json:"history_data_sharing_status,omitempty"`
	Name                              string                         `json:"name"`
	SharedAs                          string                         `json:"shared_as,omitempty"`
	StartVersion                      int                            `json:"start_version,omitempty"`
	Status                            string                         `json:"status,omitempty"`
	StringSharedAs                    string                         `json:"string_shared_as,omitempty"`
	Partition                         []ResourceShareObjectPartition `json:"partition,omitempty"`
}

type ResourceShareProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceShare struct {
	Comment         string                        `json:"comment,omitempty"`
	CreatedAt       int                           `json:"created_at,omitempty"`
	CreatedBy       string                        `json:"created_by,omitempty"`
	EffectiveOwner  string                        `json:"effective_owner,omitempty"`
	Id              string                        `json:"id,omitempty"`
	Name            string                        `json:"name"`
	Owner           string                        `json:"owner,omitempty"`
	StorageLocation string                        `json:"storage_location,omitempty"`
	StorageRoot     string                        `json:"storage_root,omitempty"`
	UpdatedAt       int                           `json:"updated_at,omitempty"`
	UpdatedBy       string                        `json:"updated_by,omitempty"`
	Object          []ResourceShareObject         `json:"object,omitempty"`
	ProviderConfig  []ResourceShareProviderConfig `json:"provider_config,omitempty"`
}
