// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresCdfStatusesCdfStatusesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresCdfStatusesCdfStatuses struct {
	CommittedLsn   string                                                  `json:"committed_lsn,omitempty"`
	CreateTime     string                                                  `json:"create_time,omitempty"`
	LastSyncTime   string                                                  `json:"last_sync_time,omitempty"`
	Name           string                                                  `json:"name"`
	PostgresTable  string                                                  `json:"postgres_table,omitempty"`
	ProviderConfig *DataSourcePostgresCdfStatusesCdfStatusesProviderConfig `json:"provider_config,omitempty"`
	State          string                                                  `json:"state,omitempty"`
	StatusDetail   string                                                  `json:"status_detail,omitempty"`
	UcTable        string                                                  `json:"uc_table,omitempty"`
}

type DataSourcePostgresCdfStatusesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresCdfStatuses struct {
	CdfStatuses    []DataSourcePostgresCdfStatusesCdfStatuses   `json:"cdf_statuses,omitempty"`
	PageSize       int                                          `json:"page_size,omitempty"`
	Parent         string                                       `json:"parent"`
	ProviderConfig *DataSourcePostgresCdfStatusesProviderConfig `json:"provider_config,omitempty"`
}
