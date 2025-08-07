// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTableColumns struct {
	Comment          string                                                                  `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTableColumnsMask `json:"mask,omitempty"`
	Name             string                                                                  `json:"name,omitempty"`
	Nullable         bool                                                                    `json:"nullable,omitempty"`
	PartitionIndex   int                                                                     `json:"partition_index,omitempty"`
	Position         int                                                                     `json:"position,omitempty"`
	TypeIntervalType string                                                                  `json:"type_interval_type,omitempty"`
	TypeJson         string                                                                  `json:"type_json,omitempty"`
	TypeName         string                                                                  `json:"type_name,omitempty"`
	TypePrecision    int                                                                     `json:"type_precision,omitempty"`
	TypeScale        int                                                                     `json:"type_scale,omitempty"`
	TypeText         string                                                                  `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTable struct {
	Columns []DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTableLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetNotebookReviews struct {
	Comment                   string `json:"comment,omitempty"`
	CreatedAtMillis           int    `json:"created_at_millis,omitempty"`
	ReviewState               string `json:"review_state,omitempty"`
	ReviewSubReason           string `json:"review_sub_reason,omitempty"`
	ReviewerCollaboratorAlias string `json:"reviewer_collaborator_alias,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetNotebook struct {
	Etag                      string                                                           `json:"etag,omitempty"`
	NotebookContent           string                                                           `json:"notebook_content"`
	ReviewState               string                                                           `json:"review_state,omitempty"`
	Reviews                   []DataSourceCleanRoomAssetRevisionsCleanRoomAssetNotebookReviews `json:"reviews,omitempty"`
	RunnerCollaboratorAliases []string                                                         `json:"runner_collaborator_aliases,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableColumns struct {
	Comment          string                                                           `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableColumnsMask `json:"mask,omitempty"`
	Name             string                                                           `json:"name,omitempty"`
	Nullable         bool                                                             `json:"nullable,omitempty"`
	PartitionIndex   int                                                              `json:"partition_index,omitempty"`
	Position         int                                                              `json:"position,omitempty"`
	TypeIntervalType string                                                           `json:"type_interval_type,omitempty"`
	TypeJson         string                                                           `json:"type_json,omitempty"`
	TypeName         string                                                           `json:"type_name,omitempty"`
	TypePrecision    int                                                              `json:"type_precision,omitempty"`
	TypeScale        int                                                              `json:"type_scale,omitempty"`
	TypeText         string                                                           `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetTable struct {
	Columns []DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableLocalDetailsPartitionsValue struct {
	Name                 string `json:"name,omitempty"`
	Op                   string `json:"op,omitempty"`
	RecipientPropertyKey string `json:"recipient_property_key,omitempty"`
	Value                string `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableLocalDetailsPartitions struct {
	Value []DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableLocalDetailsPartitionsValue `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableLocalDetails struct {
	LocalName  string                                                                       `json:"local_name"`
	Partitions []DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableLocalDetailsPartitions `json:"partitions,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetViewColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetViewColumns struct {
	Comment          string                                                          `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetViewColumnsMask `json:"mask,omitempty"`
	Name             string                                                          `json:"name,omitempty"`
	Nullable         bool                                                            `json:"nullable,omitempty"`
	PartitionIndex   int                                                             `json:"partition_index,omitempty"`
	Position         int                                                             `json:"position,omitempty"`
	TypeIntervalType string                                                          `json:"type_interval_type,omitempty"`
	TypeJson         string                                                          `json:"type_json,omitempty"`
	TypeName         string                                                          `json:"type_name,omitempty"`
	TypePrecision    int                                                             `json:"type_precision,omitempty"`
	TypeScale        int                                                             `json:"type_scale,omitempty"`
	TypeText         string                                                          `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetView struct {
	Columns []DataSourceCleanRoomAssetRevisionsCleanRoomAssetViewColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetViewLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetVolumeLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAsset struct {
	AddedAt                  int                                                                      `json:"added_at,omitempty"`
	AssetType                string                                                                   `json:"asset_type"`
	CleanRoomName            string                                                                   `json:"clean_room_name,omitempty"`
	ForeignTable             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTable             `json:"foreign_table,omitempty"`
	ForeignTableLocalDetails *DataSourceCleanRoomAssetRevisionsCleanRoomAssetForeignTableLocalDetails `json:"foreign_table_local_details,omitempty"`
	Name                     string                                                                   `json:"name"`
	Notebook                 *DataSourceCleanRoomAssetRevisionsCleanRoomAssetNotebook                 `json:"notebook,omitempty"`
	OwnerCollaboratorAlias   string                                                                   `json:"owner_collaborator_alias,omitempty"`
	Status                   string                                                                   `json:"status,omitempty"`
	Table                    *DataSourceCleanRoomAssetRevisionsCleanRoomAssetTable                    `json:"table,omitempty"`
	TableLocalDetails        *DataSourceCleanRoomAssetRevisionsCleanRoomAssetTableLocalDetails        `json:"table_local_details,omitempty"`
	View                     *DataSourceCleanRoomAssetRevisionsCleanRoomAssetView                     `json:"view,omitempty"`
	ViewLocalDetails         *DataSourceCleanRoomAssetRevisionsCleanRoomAssetViewLocalDetails         `json:"view_local_details,omitempty"`
	VolumeLocalDetails       *DataSourceCleanRoomAssetRevisionsCleanRoomAssetVolumeLocalDetails       `json:"volume_local_details,omitempty"`
}
