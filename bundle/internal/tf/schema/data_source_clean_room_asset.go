// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCleanRoomAssetForeignTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetForeignTableColumns struct {
	Comment          string                                           `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetForeignTableColumnsMask `json:"mask,omitempty"`
	Name             string                                           `json:"name,omitempty"`
	Nullable         bool                                             `json:"nullable,omitempty"`
	PartitionIndex   int                                              `json:"partition_index,omitempty"`
	Position         int                                              `json:"position,omitempty"`
	TypeIntervalType string                                           `json:"type_interval_type,omitempty"`
	TypeJson         string                                           `json:"type_json,omitempty"`
	TypeName         string                                           `json:"type_name,omitempty"`
	TypePrecision    int                                              `json:"type_precision,omitempty"`
	TypeScale        int                                              `json:"type_scale,omitempty"`
	TypeText         string                                           `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetForeignTable struct {
	Columns []DataSourceCleanRoomAssetForeignTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetForeignTableLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetNotebookReviews struct {
	Comment                   string `json:"comment,omitempty"`
	CreatedAtMillis           int    `json:"created_at_millis,omitempty"`
	ReviewState               string `json:"review_state,omitempty"`
	ReviewSubReason           string `json:"review_sub_reason,omitempty"`
	ReviewerCollaboratorAlias string `json:"reviewer_collaborator_alias,omitempty"`
}

type DataSourceCleanRoomAssetNotebook struct {
	Etag                      string                                    `json:"etag,omitempty"`
	NotebookContent           string                                    `json:"notebook_content"`
	ReviewState               string                                    `json:"review_state,omitempty"`
	Reviews                   []DataSourceCleanRoomAssetNotebookReviews `json:"reviews,omitempty"`
	RunnerCollaboratorAliases []string                                  `json:"runner_collaborator_aliases,omitempty"`
}

type DataSourceCleanRoomAssetTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetTableColumns struct {
	Comment          string                                    `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetTableColumnsMask `json:"mask,omitempty"`
	Name             string                                    `json:"name,omitempty"`
	Nullable         bool                                      `json:"nullable,omitempty"`
	PartitionIndex   int                                       `json:"partition_index,omitempty"`
	Position         int                                       `json:"position,omitempty"`
	TypeIntervalType string                                    `json:"type_interval_type,omitempty"`
	TypeJson         string                                    `json:"type_json,omitempty"`
	TypeName         string                                    `json:"type_name,omitempty"`
	TypePrecision    int                                       `json:"type_precision,omitempty"`
	TypeScale        int                                       `json:"type_scale,omitempty"`
	TypeText         string                                    `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetTable struct {
	Columns []DataSourceCleanRoomAssetTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetTableLocalDetailsPartitionsValue struct {
	Name                 string `json:"name,omitempty"`
	Op                   string `json:"op,omitempty"`
	RecipientPropertyKey string `json:"recipient_property_key,omitempty"`
	Value                string `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetTableLocalDetailsPartitions struct {
	Value []DataSourceCleanRoomAssetTableLocalDetailsPartitionsValue `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetTableLocalDetails struct {
	LocalName  string                                                `json:"local_name"`
	Partitions []DataSourceCleanRoomAssetTableLocalDetailsPartitions `json:"partitions,omitempty"`
}

type DataSourceCleanRoomAssetViewColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetViewColumns struct {
	Comment          string                                   `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetViewColumnsMask `json:"mask,omitempty"`
	Name             string                                   `json:"name,omitempty"`
	Nullable         bool                                     `json:"nullable,omitempty"`
	PartitionIndex   int                                      `json:"partition_index,omitempty"`
	Position         int                                      `json:"position,omitempty"`
	TypeIntervalType string                                   `json:"type_interval_type,omitempty"`
	TypeJson         string                                   `json:"type_json,omitempty"`
	TypeName         string                                   `json:"type_name,omitempty"`
	TypePrecision    int                                      `json:"type_precision,omitempty"`
	TypeScale        int                                      `json:"type_scale,omitempty"`
	TypeText         string                                   `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetView struct {
	Columns []DataSourceCleanRoomAssetViewColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetViewLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetVolumeLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAsset struct {
	AddedAt                  int                                               `json:"added_at,omitempty"`
	AssetType                string                                            `json:"asset_type"`
	CleanRoomName            string                                            `json:"clean_room_name,omitempty"`
	ForeignTable             *DataSourceCleanRoomAssetForeignTable             `json:"foreign_table,omitempty"`
	ForeignTableLocalDetails *DataSourceCleanRoomAssetForeignTableLocalDetails `json:"foreign_table_local_details,omitempty"`
	Name                     string                                            `json:"name"`
	Notebook                 *DataSourceCleanRoomAssetNotebook                 `json:"notebook,omitempty"`
	OwnerCollaboratorAlias   string                                            `json:"owner_collaborator_alias,omitempty"`
	Status                   string                                            `json:"status,omitempty"`
	Table                    *DataSourceCleanRoomAssetTable                    `json:"table,omitempty"`
	TableLocalDetails        *DataSourceCleanRoomAssetTableLocalDetails        `json:"table_local_details,omitempty"`
	View                     *DataSourceCleanRoomAssetView                     `json:"view,omitempty"`
	ViewLocalDetails         *DataSourceCleanRoomAssetViewLocalDetails         `json:"view_local_details,omitempty"`
	VolumeLocalDetails       *DataSourceCleanRoomAssetVolumeLocalDetails       `json:"volume_local_details,omitempty"`
}
