// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFunctionsFunctionsInputParamsParameters struct {
	Comment          string `json:"comment,omitempty"`
	Name             string `json:"name"`
	ParameterDefault string `json:"parameter_default,omitempty"`
	ParameterMode    string `json:"parameter_mode,omitempty"`
	ParameterType    string `json:"parameter_type,omitempty"`
	Position         int    `json:"position"`
	TypeIntervalType string `json:"type_interval_type,omitempty"`
	TypeJson         string `json:"type_json,omitempty"`
	TypeName         string `json:"type_name"`
	TypePrecision    int    `json:"type_precision,omitempty"`
	TypeScale        int    `json:"type_scale,omitempty"`
	TypeText         string `json:"type_text"`
}

type DataSourceFunctionsFunctionsInputParams struct {
	Parameters []DataSourceFunctionsFunctionsInputParamsParameters `json:"parameters,omitempty"`
}

type DataSourceFunctionsFunctionsReturnParamsParameters struct {
	Comment          string `json:"comment,omitempty"`
	Name             string `json:"name"`
	ParameterDefault string `json:"parameter_default,omitempty"`
	ParameterMode    string `json:"parameter_mode,omitempty"`
	ParameterType    string `json:"parameter_type,omitempty"`
	Position         int    `json:"position"`
	TypeIntervalType string `json:"type_interval_type,omitempty"`
	TypeJson         string `json:"type_json,omitempty"`
	TypeName         string `json:"type_name"`
	TypePrecision    int    `json:"type_precision,omitempty"`
	TypeScale        int    `json:"type_scale,omitempty"`
	TypeText         string `json:"type_text"`
}

type DataSourceFunctionsFunctionsReturnParams struct {
	Parameters []DataSourceFunctionsFunctionsReturnParamsParameters `json:"parameters,omitempty"`
}

type DataSourceFunctionsFunctionsRoutineDependenciesDependenciesConnection struct {
	ConnectionName string `json:"connection_name,omitempty"`
}

type DataSourceFunctionsFunctionsRoutineDependenciesDependenciesCredential struct {
	CredentialName string `json:"credential_name,omitempty"`
}

type DataSourceFunctionsFunctionsRoutineDependenciesDependenciesFunction struct {
	FunctionFullName string `json:"function_full_name"`
}

type DataSourceFunctionsFunctionsRoutineDependenciesDependenciesTable struct {
	TableFullName string `json:"table_full_name"`
}

type DataSourceFunctionsFunctionsRoutineDependenciesDependencies struct {
	Connection []DataSourceFunctionsFunctionsRoutineDependenciesDependenciesConnection `json:"connection,omitempty"`
	Credential []DataSourceFunctionsFunctionsRoutineDependenciesDependenciesCredential `json:"credential,omitempty"`
	Function   []DataSourceFunctionsFunctionsRoutineDependenciesDependenciesFunction   `json:"function,omitempty"`
	Table      []DataSourceFunctionsFunctionsRoutineDependenciesDependenciesTable      `json:"table,omitempty"`
}

type DataSourceFunctionsFunctionsRoutineDependencies struct {
	Dependencies []DataSourceFunctionsFunctionsRoutineDependenciesDependencies `json:"dependencies,omitempty"`
}

type DataSourceFunctionsFunctions struct {
	BrowseOnly          bool                                              `json:"browse_only,omitempty"`
	CatalogName         string                                            `json:"catalog_name,omitempty"`
	Comment             string                                            `json:"comment,omitempty"`
	CreatedAt           int                                               `json:"created_at,omitempty"`
	CreatedBy           string                                            `json:"created_by,omitempty"`
	DataType            string                                            `json:"data_type,omitempty"`
	ExternalLanguage    string                                            `json:"external_language,omitempty"`
	ExternalName        string                                            `json:"external_name,omitempty"`
	FullDataType        string                                            `json:"full_data_type,omitempty"`
	FullName            string                                            `json:"full_name,omitempty"`
	FunctionId          string                                            `json:"function_id,omitempty"`
	InputParams         []DataSourceFunctionsFunctionsInputParams         `json:"input_params,omitempty"`
	IsDeterministic     bool                                              `json:"is_deterministic,omitempty"`
	IsNullCall          bool                                              `json:"is_null_call,omitempty"`
	MetastoreId         string                                            `json:"metastore_id,omitempty"`
	Name                string                                            `json:"name,omitempty"`
	Owner               string                                            `json:"owner,omitempty"`
	ParameterStyle      string                                            `json:"parameter_style,omitempty"`
	Properties          string                                            `json:"properties,omitempty"`
	ReturnParams        []DataSourceFunctionsFunctionsReturnParams        `json:"return_params,omitempty"`
	RoutineBody         string                                            `json:"routine_body,omitempty"`
	RoutineDefinition   string                                            `json:"routine_definition,omitempty"`
	RoutineDependencies []DataSourceFunctionsFunctionsRoutineDependencies `json:"routine_dependencies,omitempty"`
	SchemaName          string                                            `json:"schema_name,omitempty"`
	SecurityType        string                                            `json:"security_type,omitempty"`
	SpecificName        string                                            `json:"specific_name,omitempty"`
	SqlDataAccess       string                                            `json:"sql_data_access,omitempty"`
	SqlPath             string                                            `json:"sql_path,omitempty"`
	UpdatedAt           int                                               `json:"updated_at,omitempty"`
	UpdatedBy           string                                            `json:"updated_by,omitempty"`
}

type DataSourceFunctions struct {
	CatalogName   string                         `json:"catalog_name"`
	Functions     []DataSourceFunctionsFunctions `json:"functions,omitempty"`
	IncludeBrowse bool                           `json:"include_browse,omitempty"`
	SchemaName    string                         `json:"schema_name"`
}
