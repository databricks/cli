// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceTableTableInfoColumnsMask struct {
	FunctionName     string   `json:"function_name,omitempty"`
	UsingColumnNames []string `json:"using_column_names,omitempty"`
}

type DataSourceTableTableInfoColumns struct {
	Comment          string                               `json:"comment,omitempty"`
	Name             string                               `json:"name,omitempty"`
	Nullable         bool                                 `json:"nullable,omitempty"`
	PartitionIndex   int                                  `json:"partition_index,omitempty"`
	Position         int                                  `json:"position,omitempty"`
	TypeIntervalType string                               `json:"type_interval_type,omitempty"`
	TypeJson         string                               `json:"type_json,omitempty"`
	TypeName         string                               `json:"type_name,omitempty"`
	TypePrecision    int                                  `json:"type_precision,omitempty"`
	TypeScale        int                                  `json:"type_scale,omitempty"`
	TypeText         string                               `json:"type_text,omitempty"`
	Mask             *DataSourceTableTableInfoColumnsMask `json:"mask,omitempty"`
}

type DataSourceTableTableInfoDeltaRuntimePropertiesKvpairs struct {
	DeltaRuntimeProperties map[string]string `json:"delta_runtime_properties"`
}

type DataSourceTableTableInfoEffectivePredictiveOptimizationFlag struct {
	InheritedFromName string `json:"inherited_from_name,omitempty"`
	InheritedFromType string `json:"inherited_from_type,omitempty"`
	Value             string `json:"value"`
}

type DataSourceTableTableInfoEncryptionDetailsSseEncryptionDetails struct {
	Algorithm    string `json:"algorithm,omitempty"`
	AwsKmsKeyArn string `json:"aws_kms_key_arn,omitempty"`
}

type DataSourceTableTableInfoEncryptionDetails struct {
	SseEncryptionDetails *DataSourceTableTableInfoEncryptionDetailsSseEncryptionDetails `json:"sse_encryption_details,omitempty"`
}

type DataSourceTableTableInfoRowFilter struct {
	FunctionName     string   `json:"function_name"`
	InputColumnNames []string `json:"input_column_names"`
}

type DataSourceTableTableInfoSecurableKindManifestOptions struct {
	AllowedValues []string `json:"allowed_values,omitempty"`
	DefaultValue  string   `json:"default_value,omitempty"`
	Description   string   `json:"description,omitempty"`
	Hint          string   `json:"hint,omitempty"`
	IsCopiable    bool     `json:"is_copiable,omitempty"`
	IsCreatable   bool     `json:"is_creatable,omitempty"`
	IsHidden      bool     `json:"is_hidden,omitempty"`
	IsLoggable    bool     `json:"is_loggable,omitempty"`
	IsRequired    bool     `json:"is_required,omitempty"`
	IsSecret      bool     `json:"is_secret,omitempty"`
	IsUpdatable   bool     `json:"is_updatable,omitempty"`
	Name          string   `json:"name,omitempty"`
	OauthStage    string   `json:"oauth_stage,omitempty"`
	Type          string   `json:"type,omitempty"`
}

type DataSourceTableTableInfoSecurableKindManifest struct {
	AssignablePrivileges []string                                               `json:"assignable_privileges,omitempty"`
	Capabilities         []string                                               `json:"capabilities,omitempty"`
	SecurableKind        string                                                 `json:"securable_kind,omitempty"`
	SecurableType        string                                                 `json:"securable_type,omitempty"`
	Options              []DataSourceTableTableInfoSecurableKindManifestOptions `json:"options,omitempty"`
}

type DataSourceTableTableInfoTableConstraintsForeignKeyConstraint struct {
	ChildColumns  []string `json:"child_columns"`
	Name          string   `json:"name"`
	ParentColumns []string `json:"parent_columns"`
	ParentTable   string   `json:"parent_table"`
	Rely          bool     `json:"rely,omitempty"`
}

type DataSourceTableTableInfoTableConstraintsNamedTableConstraint struct {
	Name string `json:"name"`
}

type DataSourceTableTableInfoTableConstraintsPrimaryKeyConstraint struct {
	ChildColumns      []string `json:"child_columns"`
	Name              string   `json:"name"`
	Rely              bool     `json:"rely,omitempty"`
	TimeseriesColumns []string `json:"timeseries_columns,omitempty"`
}

type DataSourceTableTableInfoTableConstraints struct {
	ForeignKeyConstraint *DataSourceTableTableInfoTableConstraintsForeignKeyConstraint `json:"foreign_key_constraint,omitempty"`
	NamedTableConstraint *DataSourceTableTableInfoTableConstraintsNamedTableConstraint `json:"named_table_constraint,omitempty"`
	PrimaryKeyConstraint *DataSourceTableTableInfoTableConstraintsPrimaryKeyConstraint `json:"primary_key_constraint,omitempty"`
}

type DataSourceTableTableInfoViewDependenciesDependenciesConnection struct {
	ConnectionName string `json:"connection_name,omitempty"`
}

type DataSourceTableTableInfoViewDependenciesDependenciesCredential struct {
	CredentialName string `json:"credential_name,omitempty"`
}

type DataSourceTableTableInfoViewDependenciesDependenciesFunction struct {
	FunctionFullName string `json:"function_full_name"`
}

type DataSourceTableTableInfoViewDependenciesDependenciesTable struct {
	TableFullName string `json:"table_full_name"`
}

type DataSourceTableTableInfoViewDependenciesDependencies struct {
	Connection *DataSourceTableTableInfoViewDependenciesDependenciesConnection `json:"connection,omitempty"`
	Credential *DataSourceTableTableInfoViewDependenciesDependenciesCredential `json:"credential,omitempty"`
	Function   *DataSourceTableTableInfoViewDependenciesDependenciesFunction   `json:"function,omitempty"`
	Table      *DataSourceTableTableInfoViewDependenciesDependenciesTable      `json:"table,omitempty"`
}

type DataSourceTableTableInfoViewDependencies struct {
	Dependencies []DataSourceTableTableInfoViewDependenciesDependencies `json:"dependencies,omitempty"`
}

type DataSourceTableTableInfo struct {
	AccessPoint                         string                                                       `json:"access_point,omitempty"`
	BrowseOnly                          bool                                                         `json:"browse_only,omitempty"`
	CatalogName                         string                                                       `json:"catalog_name,omitempty"`
	Comment                             string                                                       `json:"comment,omitempty"`
	CreatedAt                           int                                                          `json:"created_at,omitempty"`
	CreatedBy                           string                                                       `json:"created_by,omitempty"`
	DataAccessConfigurationId           string                                                       `json:"data_access_configuration_id,omitempty"`
	DataSourceFormat                    string                                                       `json:"data_source_format,omitempty"`
	DeletedAt                           int                                                          `json:"deleted_at,omitempty"`
	EnablePredictiveOptimization        string                                                       `json:"enable_predictive_optimization,omitempty"`
	FullName                            string                                                       `json:"full_name,omitempty"`
	MetastoreId                         string                                                       `json:"metastore_id,omitempty"`
	Name                                string                                                       `json:"name,omitempty"`
	Owner                               string                                                       `json:"owner,omitempty"`
	PipelineId                          string                                                       `json:"pipeline_id,omitempty"`
	Properties                          map[string]string                                            `json:"properties,omitempty"`
	SchemaName                          string                                                       `json:"schema_name,omitempty"`
	SqlPath                             string                                                       `json:"sql_path,omitempty"`
	StorageCredentialName               string                                                       `json:"storage_credential_name,omitempty"`
	StorageLocation                     string                                                       `json:"storage_location,omitempty"`
	TableId                             string                                                       `json:"table_id,omitempty"`
	TableType                           string                                                       `json:"table_type,omitempty"`
	UpdatedAt                           int                                                          `json:"updated_at,omitempty"`
	UpdatedBy                           string                                                       `json:"updated_by,omitempty"`
	ViewDefinition                      string                                                       `json:"view_definition,omitempty"`
	Columns                             []DataSourceTableTableInfoColumns                            `json:"columns,omitempty"`
	DeltaRuntimePropertiesKvpairs       *DataSourceTableTableInfoDeltaRuntimePropertiesKvpairs       `json:"delta_runtime_properties_kvpairs,omitempty"`
	EffectivePredictiveOptimizationFlag *DataSourceTableTableInfoEffectivePredictiveOptimizationFlag `json:"effective_predictive_optimization_flag,omitempty"`
	EncryptionDetails                   *DataSourceTableTableInfoEncryptionDetails                   `json:"encryption_details,omitempty"`
	RowFilter                           *DataSourceTableTableInfoRowFilter                           `json:"row_filter,omitempty"`
	SecurableKindManifest               *DataSourceTableTableInfoSecurableKindManifest               `json:"securable_kind_manifest,omitempty"`
	TableConstraints                    []DataSourceTableTableInfoTableConstraints                   `json:"table_constraints,omitempty"`
	ViewDependencies                    *DataSourceTableTableInfoViewDependencies                    `json:"view_dependencies,omitempty"`
}

type DataSourceTable struct {
	Id        string                    `json:"id,omitempty"`
	Name      string                    `json:"name"`
	TableInfo *DataSourceTableTableInfo `json:"table_info,omitempty"`
}
