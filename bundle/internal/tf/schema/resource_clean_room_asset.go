// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceCleanRoomAssetForeignTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type ResourceCleanRoomAssetForeignTableColumns struct {
	Comment          string                                         `json:"comment,omitempty"`
	Mask             *ResourceCleanRoomAssetForeignTableColumnsMask `json:"mask,omitempty"`
	Name             string                                         `json:"name,omitempty"`
	Nullable         bool                                           `json:"nullable,omitempty"`
	PartitionIndex   int                                            `json:"partition_index,omitempty"`
	Position         int                                            `json:"position,omitempty"`
	TypeIntervalType string                                         `json:"type_interval_type,omitempty"`
	TypeJson         string                                         `json:"type_json,omitempty"`
	TypeName         string                                         `json:"type_name,omitempty"`
	TypePrecision    int                                            `json:"type_precision,omitempty"`
	TypeScale        int                                            `json:"type_scale,omitempty"`
	TypeText         string                                         `json:"type_text,omitempty"`
}

type ResourceCleanRoomAssetForeignTable struct {
	Columns []ResourceCleanRoomAssetForeignTableColumns `json:"columns,omitempty"`
}

type ResourceCleanRoomAssetForeignTableLocalDetails struct {
	LocalName string `json:"local_name"`
}

type ResourceCleanRoomAssetNotebookReviews struct {
	Comment                   string `json:"comment,omitempty"`
	CreatedAtMillis           int    `json:"created_at_millis,omitempty"`
	ReviewState               string `json:"review_state,omitempty"`
	ReviewSubReason           string `json:"review_sub_reason,omitempty"`
	ReviewerCollaboratorAlias string `json:"reviewer_collaborator_alias,omitempty"`
}

type ResourceCleanRoomAssetNotebook struct {
	Etag                      string                                  `json:"etag,omitempty"`
	NotebookContent           string                                  `json:"notebook_content"`
	ReviewState               string                                  `json:"review_state,omitempty"`
	Reviews                   []ResourceCleanRoomAssetNotebookReviews `json:"reviews,omitempty"`
	RunnerCollaboratorAliases []string                                `json:"runner_collaborator_aliases,omitempty"`
}

type ResourceCleanRoomAssetTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type ResourceCleanRoomAssetTableColumns struct {
	Comment          string                                  `json:"comment,omitempty"`
	Mask             *ResourceCleanRoomAssetTableColumnsMask `json:"mask,omitempty"`
	Name             string                                  `json:"name,omitempty"`
	Nullable         bool                                    `json:"nullable,omitempty"`
	PartitionIndex   int                                     `json:"partition_index,omitempty"`
	Position         int                                     `json:"position,omitempty"`
	TypeIntervalType string                                  `json:"type_interval_type,omitempty"`
	TypeJson         string                                  `json:"type_json,omitempty"`
	TypeName         string                                  `json:"type_name,omitempty"`
	TypePrecision    int                                     `json:"type_precision,omitempty"`
	TypeScale        int                                     `json:"type_scale,omitempty"`
	TypeText         string                                  `json:"type_text,omitempty"`
}

type ResourceCleanRoomAssetTable struct {
	Columns []ResourceCleanRoomAssetTableColumns `json:"columns,omitempty"`
}

type ResourceCleanRoomAssetTableLocalDetailsPartitionsValue struct {
	Name                 string `json:"name,omitempty"`
	Op                   string `json:"op,omitempty"`
	RecipientPropertyKey string `json:"recipient_property_key,omitempty"`
	Value                string `json:"value,omitempty"`
}

type ResourceCleanRoomAssetTableLocalDetailsPartitions struct {
	Value []ResourceCleanRoomAssetTableLocalDetailsPartitionsValue `json:"value,omitempty"`
}

type ResourceCleanRoomAssetTableLocalDetails struct {
	LocalName  string                                              `json:"local_name"`
	Partitions []ResourceCleanRoomAssetTableLocalDetailsPartitions `json:"partitions,omitempty"`
}

type ResourceCleanRoomAssetViewColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type ResourceCleanRoomAssetViewColumns struct {
	Comment          string                                 `json:"comment,omitempty"`
	Mask             *ResourceCleanRoomAssetViewColumnsMask `json:"mask,omitempty"`
	Name             string                                 `json:"name,omitempty"`
	Nullable         bool                                   `json:"nullable,omitempty"`
	PartitionIndex   int                                    `json:"partition_index,omitempty"`
	Position         int                                    `json:"position,omitempty"`
	TypeIntervalType string                                 `json:"type_interval_type,omitempty"`
	TypeJson         string                                 `json:"type_json,omitempty"`
	TypeName         string                                 `json:"type_name,omitempty"`
	TypePrecision    int                                    `json:"type_precision,omitempty"`
	TypeScale        int                                    `json:"type_scale,omitempty"`
	TypeText         string                                 `json:"type_text,omitempty"`
}

type ResourceCleanRoomAssetView struct {
	Columns []ResourceCleanRoomAssetViewColumns `json:"columns,omitempty"`
}

type ResourceCleanRoomAssetViewLocalDetails struct {
	LocalName string `json:"local_name"`
}

type ResourceCleanRoomAssetVolumeLocalDetails struct {
	LocalName string `json:"local_name"`
}

type ResourceCleanRoomAsset struct {
	AddedAt                  int                                             `json:"added_at,omitempty"`
	AssetType                string                                          `json:"asset_type"`
	CleanRoomName            string                                          `json:"clean_room_name,omitempty"`
	ForeignTable             *ResourceCleanRoomAssetForeignTable             `json:"foreign_table,omitempty"`
	ForeignTableLocalDetails *ResourceCleanRoomAssetForeignTableLocalDetails `json:"foreign_table_local_details,omitempty"`
	Name                     string                                          `json:"name"`
	Notebook                 *ResourceCleanRoomAssetNotebook                 `json:"notebook,omitempty"`
	OwnerCollaboratorAlias   string                                          `json:"owner_collaborator_alias,omitempty"`
	Status                   string                                          `json:"status,omitempty"`
	Table                    *ResourceCleanRoomAssetTable                    `json:"table,omitempty"`
	TableLocalDetails        *ResourceCleanRoomAssetTableLocalDetails        `json:"table_local_details,omitempty"`
	View                     *ResourceCleanRoomAssetView                     `json:"view,omitempty"`
	ViewLocalDetails         *ResourceCleanRoomAssetViewLocalDetails         `json:"view_local_details,omitempty"`
	VolumeLocalDetails       *ResourceCleanRoomAssetVolumeLocalDetails       `json:"volume_local_details,omitempty"`
}
