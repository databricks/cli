// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresCdfStatusProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresCdfStatus struct {
	CommittedLsn   string                                     `json:"committed_lsn,omitempty"`
	CreateTime     string                                     `json:"create_time,omitempty"`
	LastSyncTime   string                                     `json:"last_sync_time,omitempty"`
	Name           string                                     `json:"name"`
	PostgresTable  string                                     `json:"postgres_table,omitempty"`
	ProviderConfig *DataSourcePostgresCdfStatusProviderConfig `json:"provider_config,omitempty"`
	State          string                                     `json:"state,omitempty"`
	StatusDetail   string                                     `json:"status_detail,omitempty"`
	UcTable        string                                     `json:"uc_table,omitempty"`
}
