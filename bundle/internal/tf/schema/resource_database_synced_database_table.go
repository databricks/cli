// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusContinuousUpdateStatusInitialPipelineSyncProgress struct {
	EstimatedCompletionTimeSeconds   int    `json:"estimated_completion_time_seconds,omitempty"`
	LatestVersionCurrentlyProcessing int    `json:"latest_version_currently_processing,omitempty"`
	ProvisioningPhase                string `json:"provisioning_phase,omitempty"`
	SyncProgressCompletion           int    `json:"sync_progress_completion,omitempty"`
	SyncedRowCount                   int    `json:"synced_row_count,omitempty"`
	TotalRowCount                    int    `json:"total_row_count,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusContinuousUpdateStatus struct {
	InitialPipelineSyncProgress *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusContinuousUpdateStatusInitialPipelineSyncProgress `json:"initial_pipeline_sync_progress,omitempty"`
	LastProcessedCommitVersion  int                                                                                                            `json:"last_processed_commit_version,omitempty"`
	Timestamp                   string                                                                                                         `json:"timestamp,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusFailedStatus struct {
	LastProcessedCommitVersion int    `json:"last_processed_commit_version,omitempty"`
	Timestamp                  string `json:"timestamp,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusLastSyncDeltaTableSyncInfo struct {
	DeltaCommitTimestamp string `json:"delta_commit_timestamp,omitempty"`
	DeltaCommitVersion   int    `json:"delta_commit_version,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusLastSync struct {
	DeltaTableSyncInfo *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusLastSyncDeltaTableSyncInfo `json:"delta_table_sync_info,omitempty"`
	SyncEndTimestamp   string                                                                                  `json:"sync_end_timestamp,omitempty"`
	SyncStartTimestamp string                                                                                  `json:"sync_start_timestamp,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusProvisioningStatusInitialPipelineSyncProgress struct {
	EstimatedCompletionTimeSeconds   int    `json:"estimated_completion_time_seconds,omitempty"`
	LatestVersionCurrentlyProcessing int    `json:"latest_version_currently_processing,omitempty"`
	ProvisioningPhase                string `json:"provisioning_phase,omitempty"`
	SyncProgressCompletion           int    `json:"sync_progress_completion,omitempty"`
	SyncedRowCount                   int    `json:"synced_row_count,omitempty"`
	TotalRowCount                    int    `json:"total_row_count,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusProvisioningStatus struct {
	InitialPipelineSyncProgress *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusProvisioningStatusInitialPipelineSyncProgress `json:"initial_pipeline_sync_progress,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusTriggeredUpdateStatusTriggeredUpdateProgress struct {
	EstimatedCompletionTimeSeconds   int    `json:"estimated_completion_time_seconds,omitempty"`
	LatestVersionCurrentlyProcessing int    `json:"latest_version_currently_processing,omitempty"`
	ProvisioningPhase                string `json:"provisioning_phase,omitempty"`
	SyncProgressCompletion           int    `json:"sync_progress_completion,omitempty"`
	SyncedRowCount                   int    `json:"synced_row_count,omitempty"`
	TotalRowCount                    int    `json:"total_row_count,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusTriggeredUpdateStatus struct {
	LastProcessedCommitVersion int                                                                                                       `json:"last_processed_commit_version,omitempty"`
	Timestamp                  string                                                                                                    `json:"timestamp,omitempty"`
	TriggeredUpdateProgress    *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusTriggeredUpdateStatusTriggeredUpdateProgress `json:"triggered_update_progress,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatus struct {
	ContinuousUpdateStatus *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusContinuousUpdateStatus `json:"continuous_update_status,omitempty"`
	DetailedState          string                                                                              `json:"detailed_state,omitempty"`
	FailedStatus           *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusFailedStatus           `json:"failed_status,omitempty"`
	LastSync               *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusLastSync               `json:"last_sync,omitempty"`
	Message                string                                                                              `json:"message,omitempty"`
	PipelineId             string                                                                              `json:"pipeline_id,omitempty"`
	ProvisioningStatus     *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusProvisioningStatus     `json:"provisioning_status,omitempty"`
	TriggeredUpdateStatus  *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatusTriggeredUpdateStatus  `json:"triggered_update_status,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableSpecNewPipelineSpec struct {
	StorageCatalog string `json:"storage_catalog,omitempty"`
	StorageSchema  string `json:"storage_schema,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTableSpec struct {
	CreateDatabaseObjectsIfMissing bool                                                    `json:"create_database_objects_if_missing,omitempty"`
	ExistingPipelineId             string                                                  `json:"existing_pipeline_id,omitempty"`
	NewPipelineSpec                *ResourceDatabaseSyncedDatabaseTableSpecNewPipelineSpec `json:"new_pipeline_spec,omitempty"`
	PrimaryKeyColumns              []string                                                `json:"primary_key_columns,omitempty"`
	SchedulingPolicy               string                                                  `json:"scheduling_policy,omitempty"`
	SourceTableFullName            string                                                  `json:"source_table_full_name,omitempty"`
	TimeseriesKey                  string                                                  `json:"timeseries_key,omitempty"`
}

type ResourceDatabaseSyncedDatabaseTable struct {
	DataSynchronizationStatus     *ResourceDatabaseSyncedDatabaseTableDataSynchronizationStatus `json:"data_synchronization_status,omitempty"`
	DatabaseInstanceName          string                                                        `json:"database_instance_name,omitempty"`
	EffectiveDatabaseInstanceName string                                                        `json:"effective_database_instance_name,omitempty"`
	EffectiveLogicalDatabaseName  string                                                        `json:"effective_logical_database_name,omitempty"`
	LogicalDatabaseName           string                                                        `json:"logical_database_name,omitempty"`
	Name                          string                                                        `json:"name"`
	Spec                          *ResourceDatabaseSyncedDatabaseTableSpec                      `json:"spec,omitempty"`
	UnityCatalogProvisioningState string                                                        `json:"unity_catalog_provisioning_state,omitempty"`
}
