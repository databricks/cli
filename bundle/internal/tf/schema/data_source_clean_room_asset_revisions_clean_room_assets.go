// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTableColumns struct {
	Comment          string                                                                            `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTableColumnsMask `json:"mask,omitempty"`
	Name             string                                                                            `json:"name,omitempty"`
	Nullable         bool                                                                              `json:"nullable,omitempty"`
	PartitionIndex   int                                                                               `json:"partition_index,omitempty"`
	Position         int                                                                               `json:"position,omitempty"`
	TypeIntervalType string                                                                            `json:"type_interval_type,omitempty"`
	TypeJson         string                                                                            `json:"type_json,omitempty"`
	TypeName         string                                                                            `json:"type_name,omitempty"`
	TypePrecision    int                                                                               `json:"type_precision,omitempty"`
	TypeScale        int                                                                               `json:"type_scale,omitempty"`
	TypeText         string                                                                            `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTable struct {
	Columns []DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTableLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsNotebookReviews struct {
	Comment                   string `json:"comment,omitempty"`
	CreatedAtMillis           int    `json:"created_at_millis,omitempty"`
	ReviewState               string `json:"review_state,omitempty"`
	ReviewSubReason           string `json:"review_sub_reason,omitempty"`
	ReviewerCollaboratorAlias string `json:"reviewer_collaborator_alias,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsNotebook struct {
	Etag                      string                                                                     `json:"etag,omitempty"`
	NotebookContent           string                                                                     `json:"notebook_content"`
	ReviewState               string                                                                     `json:"review_state,omitempty"`
	Reviews                   []DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsNotebookReviews `json:"reviews,omitempty"`
	RunnerCollaboratorAliases []string                                                                   `json:"runner_collaborator_aliases,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableColumns struct {
	Comment          string                                                                     `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableColumnsMask `json:"mask,omitempty"`
	Name             string                                                                     `json:"name,omitempty"`
	Nullable         bool                                                                       `json:"nullable,omitempty"`
	PartitionIndex   int                                                                        `json:"partition_index,omitempty"`
	Position         int                                                                        `json:"position,omitempty"`
	TypeIntervalType string                                                                     `json:"type_interval_type,omitempty"`
	TypeJson         string                                                                     `json:"type_json,omitempty"`
	TypeName         string                                                                     `json:"type_name,omitempty"`
	TypePrecision    int                                                                        `json:"type_precision,omitempty"`
	TypeScale        int                                                                        `json:"type_scale,omitempty"`
	TypeText         string                                                                     `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTable struct {
	Columns []DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableLocalDetailsPartitionsValue struct {
	Name                 string `json:"name,omitempty"`
	Op                   string `json:"op,omitempty"`
	RecipientPropertyKey string `json:"recipient_property_key,omitempty"`
	Value                string `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableLocalDetailsPartitions struct {
	Value []DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableLocalDetailsPartitionsValue `json:"value,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableLocalDetails struct {
	LocalName  string                                                                                 `json:"local_name"`
	Partitions []DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableLocalDetailsPartitions `json:"partitions,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsViewColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsViewColumns struct {
	Comment          string                                                                    `json:"comment,omitempty"`
	Mask             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsViewColumnsMask `json:"mask,omitempty"`
	Name             string                                                                    `json:"name,omitempty"`
	Nullable         bool                                                                      `json:"nullable,omitempty"`
	PartitionIndex   int                                                                       `json:"partition_index,omitempty"`
	Position         int                                                                       `json:"position,omitempty"`
	TypeIntervalType string                                                                    `json:"type_interval_type,omitempty"`
	TypeJson         string                                                                    `json:"type_json,omitempty"`
	TypeName         string                                                                    `json:"type_name,omitempty"`
	TypePrecision    int                                                                       `json:"type_precision,omitempty"`
	TypeScale        int                                                                       `json:"type_scale,omitempty"`
	TypeText         string                                                                    `json:"type_text,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsView struct {
	Columns []DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsViewColumns `json:"columns,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsViewLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsVolumeLocalDetails struct {
	LocalName string `json:"local_name"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisions struct {
	AddedAt                  int                                                                                `json:"added_at,omitempty"`
	AssetType                string                                                                             `json:"asset_type"`
	CleanRoomName            string                                                                             `json:"clean_room_name,omitempty"`
	ForeignTable             *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTable             `json:"foreign_table,omitempty"`
	ForeignTableLocalDetails *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsForeignTableLocalDetails `json:"foreign_table_local_details,omitempty"`
	Name                     string                                                                             `json:"name"`
	Notebook                 *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsNotebook                 `json:"notebook,omitempty"`
	OwnerCollaboratorAlias   string                                                                             `json:"owner_collaborator_alias,omitempty"`
	Status                   string                                                                             `json:"status,omitempty"`
	Table                    *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTable                    `json:"table,omitempty"`
	TableLocalDetails        *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsTableLocalDetails        `json:"table_local_details,omitempty"`
	View                     *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsView                     `json:"view,omitempty"`
	ViewLocalDetails         *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsViewLocalDetails         `json:"view_local_details,omitempty"`
	VolumeLocalDetails       *DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisionsVolumeLocalDetails       `json:"volume_local_details,omitempty"`
}

type DataSourceCleanRoomAssetRevisionsCleanRoomAssets struct {
	Revisions []DataSourceCleanRoomAssetRevisionsCleanRoomAssetsRevisions `json:"revisions,omitempty"`
}
