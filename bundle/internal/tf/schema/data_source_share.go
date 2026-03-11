// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceShareObjectPartitionValue struct {
	Name                 string `json:"name,omitempty"`
	Op                   string `json:"op,omitempty"`
	RecipientPropertyKey string `json:"recipient_property_key,omitempty"`
	Value                string `json:"value,omitempty"`
}

type DataSourceShareObjectPartition struct {
	Value []DataSourceShareObjectPartitionValue `json:"value,omitempty"`
}

type DataSourceShareObject struct {
	AddedAt                           int                              `json:"added_at,omitempty"`
	AddedBy                           string                           `json:"added_by,omitempty"`
	CdfEnabled                        bool                             `json:"cdf_enabled,omitempty"`
	Comment                           string                           `json:"comment,omitempty"`
	Content                           string                           `json:"content,omitempty"`
	DataObjectType                    string                           `json:"data_object_type,omitempty"`
	EffectiveCdfEnabled               bool                             `json:"effective_cdf_enabled,omitempty"`
	EffectiveHistoryDataSharingStatus string                           `json:"effective_history_data_sharing_status,omitempty"`
	EffectiveSharedAs                 string                           `json:"effective_shared_as,omitempty"`
	EffectiveStartVersion             int                              `json:"effective_start_version,omitempty"`
	EffectiveStringSharedAs           string                           `json:"effective_string_shared_as,omitempty"`
	HistoryDataSharingStatus          string                           `json:"history_data_sharing_status,omitempty"`
	Name                              string                           `json:"name"`
	Partition                         []DataSourceShareObjectPartition `json:"partition,omitempty"`
	SharedAs                          string                           `json:"shared_as,omitempty"`
	StartVersion                      int                              `json:"start_version,omitempty"`
	Status                            string                           `json:"status,omitempty"`
	StringSharedAs                    string                           `json:"string_shared_as,omitempty"`
}

type DataSourceShareProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceShare struct {
	Comment         string                         `json:"comment,omitempty"`
	CreatedAt       int                            `json:"created_at,omitempty"`
	CreatedBy       string                         `json:"created_by,omitempty"`
	EffectiveOwner  string                         `json:"effective_owner,omitempty"`
	Name            string                         `json:"name,omitempty"`
	Object          []DataSourceShareObject        `json:"object,omitempty"`
	Owner           string                         `json:"owner,omitempty"`
	ProviderConfig  *DataSourceShareProviderConfig `json:"provider_config,omitempty"`
	StorageLocation string                         `json:"storage_location,omitempty"`
	StorageRoot     string                         `json:"storage_root,omitempty"`
	UpdatedAt       int                            `json:"updated_at,omitempty"`
	UpdatedBy       string                         `json:"updated_by,omitempty"`
}
