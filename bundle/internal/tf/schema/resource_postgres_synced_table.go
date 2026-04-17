// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresSyncedTableProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourcePostgresSyncedTableSpecNewPipelineSpec struct {
	BudgetPolicyId string `json:"budget_policy_id,omitempty"`
	StorageCatalog string `json:"storage_catalog,omitempty"`
	StorageSchema  string `json:"storage_schema,omitempty"`
}

type ResourcePostgresSyncedTableSpec struct {
	Branch                         string                                          `json:"branch,omitempty"`
	CreateDatabaseObjectsIfMissing bool                                            `json:"create_database_objects_if_missing,omitempty"`
	ExistingPipelineId             string                                          `json:"existing_pipeline_id,omitempty"`
	NewPipelineSpec                *ResourcePostgresSyncedTableSpecNewPipelineSpec `json:"new_pipeline_spec,omitempty"`
	PostgresDatabase               string                                          `json:"postgres_database,omitempty"`
	PrimaryKeyColumns              []string                                        `json:"primary_key_columns,omitempty"`
	SchedulingPolicy               string                                          `json:"scheduling_policy,omitempty"`
	SourceTableFullName            string                                          `json:"source_table_full_name,omitempty"`
	TimeseriesKey                  string                                          `json:"timeseries_key,omitempty"`
}

type ResourcePostgresSyncedTableStatusLastSyncDeltaTableSyncInfo struct {
	DeltaCommitTime    string `json:"delta_commit_time,omitempty"`
	DeltaCommitVersion int    `json:"delta_commit_version,omitempty"`
}

type ResourcePostgresSyncedTableStatusLastSync struct {
	DeltaTableSyncInfo *ResourcePostgresSyncedTableStatusLastSyncDeltaTableSyncInfo `json:"delta_table_sync_info,omitempty"`
	SyncEndTime        string                                                       `json:"sync_end_time,omitempty"`
	SyncStartTime      string                                                       `json:"sync_start_time,omitempty"`
}

type ResourcePostgresSyncedTableStatusOngoingSyncProgress struct {
	EstimatedCompletionTimeSeconds   int `json:"estimated_completion_time_seconds,omitempty"`
	LatestVersionCurrentlyProcessing int `json:"latest_version_currently_processing,omitempty"`
	SyncProgressCompletion           int `json:"sync_progress_completion,omitempty"`
	SyncedRowCount                   int `json:"synced_row_count,omitempty"`
	TotalRowCount                    int `json:"total_row_count,omitempty"`
}

type ResourcePostgresSyncedTableStatus struct {
	DetailedState                 string                                                `json:"detailed_state,omitempty"`
	LastProcessedCommitVersion    int                                                   `json:"last_processed_commit_version,omitempty"`
	LastSync                      *ResourcePostgresSyncedTableStatusLastSync            `json:"last_sync,omitempty"`
	LastSyncTime                  string                                                `json:"last_sync_time,omitempty"`
	Message                       string                                                `json:"message,omitempty"`
	OngoingSyncProgress           *ResourcePostgresSyncedTableStatusOngoingSyncProgress `json:"ongoing_sync_progress,omitempty"`
	PipelineId                    string                                                `json:"pipeline_id,omitempty"`
	ProvisioningPhase             string                                                `json:"provisioning_phase,omitempty"`
	UnityCatalogProvisioningState string                                                `json:"unity_catalog_provisioning_state,omitempty"`
}

type ResourcePostgresSyncedTable struct {
	CreateTime     string                                     `json:"create_time,omitempty"`
	Name           string                                     `json:"name,omitempty"`
	ProviderConfig *ResourcePostgresSyncedTableProviderConfig `json:"provider_config,omitempty"`
	Spec           *ResourcePostgresSyncedTableSpec           `json:"spec,omitempty"`
	Status         *ResourcePostgresSyncedTableStatus         `json:"status,omitempty"`
	SyncedTableId  string                                     `json:"synced_table_id"`
	Uid            string                                     `json:"uid,omitempty"`
}
