// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCleanRoomAssetsAssetsForeignTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsForeignTableColumns struct {
	Comment          string                                                  `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetsAssetsForeignTableColumnsMask `json:"mask,omitempty"`
	Name             string                                                  `json:"name,omitempty"`
	Nullable         bool                                                    `json:"nullable,omitempty"`
	PartitionIndex   int                                                     `json:"partition_index,omitempty"`
	Position         int                                                     `json:"position,omitempty"`
	TypeIntervalType string                                                  `json:"type_interval_type,omitempty"`
	TypeJson         string                                                  `json:"type_json,omitempty"`
	TypeName         string                                                  `json:"type_name,omitempty"`
	TypePrecision    int                                                     `json:"type_precision,omitempty"`
	TypeScale        int                                                     `json:"type_scale,omitempty"`
	TypeText         string                                                  `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsForeignTable struct {
	Columns []DataSourceCleanRoomAssetsAssetsForeignTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsForeignTableLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetsAssetsNotebookReviews struct {
	Comment                   string `json:"comment,omitempty"`
	CreatedAtMillis           int    `json:"created_at_millis,omitempty"`
	ReviewState               string `json:"review_state,omitempty"`
	ReviewSubReason           string `json:"review_sub_reason,omitempty"`
	ReviewerCollaboratorAlias string `json:"reviewer_collaborator_alias,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsNotebook struct {
	Etag                      string                                           `json:"etag,omitempty"`
	NotebookContent           string                                           `json:"notebook_content"`
	ReviewState               string                                           `json:"review_state,omitempty"`
	Reviews                   []DataSourceCleanRoomAssetsAssetsNotebookReviews `json:"reviews,omitempty"`
	RunnerCollaboratorAliases []string                                         `json:"runner_collaborator_aliases,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsTableColumns struct {
	Comment          string                                           `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetsAssetsTableColumnsMask `json:"mask,omitempty"`
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

type DataSourceCleanRoomAssetsAssetsTable struct {
	Columns []DataSourceCleanRoomAssetsAssetsTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsTableLocalDetailsPartitionsValue struct {
	Name                 string `json:"name,omitempty"`
	Op                   string `json:"op,omitempty"`
	RecipientPropertyKey string `json:"recipient_property_key,omitempty"`
	Value                string `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsTableLocalDetailsPartitions struct {
	Value []DataSourceCleanRoomAssetsAssetsTableLocalDetailsPartitionsValue `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsTableLocalDetails struct {
	LocalName  string                                                       `json:"local_name"`
	Partitions []DataSourceCleanRoomAssetsAssetsTableLocalDetailsPartitions `json:"partitions,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsViewColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsViewColumns struct {
	Comment          string                                          `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetsAssetsViewColumnsMask `json:"mask,omitempty"`
	Name             string                                          `json:"name,omitempty"`
	Nullable         bool                                            `json:"nullable,omitempty"`
	PartitionIndex   int                                             `json:"partition_index,omitempty"`
	Position         int                                             `json:"position,omitempty"`
	TypeIntervalType string                                          `json:"type_interval_type,omitempty"`
	TypeJson         string                                          `json:"type_json,omitempty"`
	TypeName         string                                          `json:"type_name,omitempty"`
	TypePrecision    int                                             `json:"type_precision,omitempty"`
	TypeScale        int                                             `json:"type_scale,omitempty"`
	TypeText         string                                          `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsView struct {
	Columns []DataSourceCleanRoomAssetsAssetsViewColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetsAssetsViewLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetsAssetsVolumeLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetsAssets struct {
	AddedAt                  int                                                      `json:"added_at,omitempty"`
	AssetType                string                                                   `json:"asset_type"`
	CleanRoomName            string                                                   `json:"clean_room_name,omitempty"`
	ForeignTable             *DataSourceCleanRoomAssetsAssetsForeignTable             `json:"foreign_table,omitempty"`
	ForeignTableLocalDetails *DataSourceCleanRoomAssetsAssetsForeignTableLocalDetails `json:"foreign_table_local_details,omitempty"`
	Name                     string                                                   `json:"name"`
	Notebook                 *DataSourceCleanRoomAssetsAssetsNotebook                 `json:"notebook,omitempty"`
	OwnerCollaboratorAlias   string                                                   `json:"owner_collaborator_alias,omitempty"`
	Status                   string                                                   `json:"status,omitempty"`
	Table                    *DataSourceCleanRoomAssetsAssetsTable                    `json:"table,omitempty"`
	TableLocalDetails        *DataSourceCleanRoomAssetsAssetsTableLocalDetails        `json:"table_local_details,omitempty"`
	View                     *DataSourceCleanRoomAssetsAssetsView                     `json:"view,omitempty"`
	ViewLocalDetails         *DataSourceCleanRoomAssetsAssetsViewLocalDetails         `json:"view_local_details,omitempty"`
	VolumeLocalDetails       *DataSourceCleanRoomAssetsAssetsVolumeLocalDetails       `json:"volume_local_details,omitempty"`
}

type DataSourceCleanRoomAssets struct {
	Assets []DataSourceCleanRoomAssetsAssets `json:"assets,omitempty"`
}
