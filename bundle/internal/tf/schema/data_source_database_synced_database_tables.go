// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusContinuousUpdateStatusInitialPipelineSyncProgress struct {
	EstimatedCompletionTimeSeconds   int    `json:"estimated_completion_time_seconds,omitempty"`
	LatestVersionCurrentlyProcessing int    `json:"latest_version_currently_processing,omitempty"`
	ProvisioningPhase                string `json:"provisioning_phase,omitempty"`
	SyncProgressCompletion           int    `json:"sync_progress_completion,omitempty"`
	SyncedRowCount                   int    `json:"synced_row_count,omitempty"`
	TotalRowCount                    int    `json:"total_row_count,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusContinuousUpdateStatus struct {
	InitialPipelineSyncProgress *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusContinuousUpdateStatusInitialPipelineSyncProgress `json:"initial_pipeline_sync_progress,omitempty"`
	LastProcessedCommitVersion  int                                                                                                                           `json:"last_processed_commit_version,omitempty"`
	Timestamp                   string                                                                                                                        `json:"timestamp,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusFailedStatus struct {
	LastProcessedCommitVersion int    `json:"last_processed_commit_version,omitempty"`
	Timestamp                  string `json:"timestamp,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusLastSyncDeltaTableSyncInfo struct {
	DeltaCommitTimestamp string `json:"delta_commit_timestamp,omitempty"`
	DeltaCommitVersion   int    `json:"delta_commit_version,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusLastSync struct {
	DeltaTableSyncInfo *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusLastSyncDeltaTableSyncInfo `json:"delta_table_sync_info,omitempty"`
	SyncEndTimestamp   string                                                                                                 `json:"sync_end_timestamp,omitempty"`
	SyncStartTimestamp string                                                                                                 `json:"sync_start_timestamp,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusProvisioningStatusInitialPipelineSyncProgress struct {
	EstimatedCompletionTimeSeconds   int    `json:"estimated_completion_time_seconds,omitempty"`
	LatestVersionCurrentlyProcessing int    `json:"latest_version_currently_processing,omitempty"`
	ProvisioningPhase                string `json:"provisioning_phase,omitempty"`
	SyncProgressCompletion           int    `json:"sync_progress_completion,omitempty"`
	SyncedRowCount                   int    `json:"synced_row_count,omitempty"`
	TotalRowCount                    int    `json:"total_row_count,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusProvisioningStatus struct {
	InitialPipelineSyncProgress *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusProvisioningStatusInitialPipelineSyncProgress `json:"initial_pipeline_sync_progress,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusTriggeredUpdateStatusTriggeredUpdateProgress struct {
	EstimatedCompletionTimeSeconds   int    `json:"estimated_completion_time_seconds,omitempty"`
	LatestVersionCurrentlyProcessing int    `json:"latest_version_currently_processing,omitempty"`
	ProvisioningPhase                string `json:"provisioning_phase,omitempty"`
	SyncProgressCompletion           int    `json:"sync_progress_completion,omitempty"`
	SyncedRowCount                   int    `json:"synced_row_count,omitempty"`
	TotalRowCount                    int    `json:"total_row_count,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusTriggeredUpdateStatus struct {
	LastProcessedCommitVersion int                                                                                                                      `json:"last_processed_commit_version,omitempty"`
	Timestamp                  string                                                                                                                   `json:"timestamp,omitempty"`
	TriggeredUpdateProgress    *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusTriggeredUpdateStatusTriggeredUpdateProgress `json:"triggered_update_progress,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatus struct {
	ContinuousUpdateStatus *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusContinuousUpdateStatus `json:"continuous_update_status,omitempty"`
	DetailedState          string                                                                                             `json:"detailed_state,omitempty"`
	FailedStatus           *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusFailedStatus           `json:"failed_status,omitempty"`
	LastSync               *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusLastSync               `json:"last_sync,omitempty"`
	Message                string                                                                                             `json:"message,omitempty"`
	PipelineId             string                                                                                             `json:"pipeline_id,omitempty"`
	ProvisioningStatus     *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusProvisioningStatus     `json:"provisioning_status,omitempty"`
	TriggeredUpdateStatus  *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatusTriggeredUpdateStatus  `json:"triggered_update_status,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesSpecNewPipelineSpec struct {
	StorageCatalog string `json:"storage_catalog,omitempty"`
	StorageSchema  string `json:"storage_schema,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTablesSpec struct {
	CreateDatabaseObjectsIfMissing bool                                                                   `json:"create_database_objects_if_missing,omitempty"`
	ExistingPipelineId             string                                                                 `json:"existing_pipeline_id,omitempty"`
	NewPipelineSpec                *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesSpecNewPipelineSpec `json:"new_pipeline_spec,omitempty"`
	PrimaryKeyColumns              []string                                                               `json:"primary_key_columns,omitempty"`
	SchedulingPolicy               string                                                                 `json:"scheduling_policy,omitempty"`
	SourceTableFullName            string                                                                 `json:"source_table_full_name,omitempty"`
	TimeseriesKey                  string                                                                 `json:"timeseries_key,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTablesSyncedTables struct {
	DataSynchronizationStatus     *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesDataSynchronizationStatus `json:"data_synchronization_status,omitempty"`
	DatabaseInstanceName          string                                                                       `json:"database_instance_name,omitempty"`
	EffectiveDatabaseInstanceName string                                                                       `json:"effective_database_instance_name,omitempty"`
	EffectiveLogicalDatabaseName  string                                                                       `json:"effective_logical_database_name,omitempty"`
	LogicalDatabaseName           string                                                                       `json:"logical_database_name,omitempty"`
	Name                          string                                                                       `json:"name"`
	Spec                          *DataSourceDatabaseSyncedDatabaseTablesSyncedTablesSpec                      `json:"spec,omitempty"`
	UnityCatalogProvisioningState string                                                                       `json:"unity_catalog_provisioning_state,omitempty"`
}

type DataSourceDatabaseSyncedDatabaseTables struct {
	SyncedTables []DataSourceDatabaseSyncedDatabaseTablesSyncedTables `json:"synced_tables,omitempty"`
	WorkspaceId  string                                               `json:"workspace_id,omitempty"`
}
