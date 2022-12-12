// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceTableColumn struct {
	Comment          string `json:"comment,omitempty"`
	Name             string `json:"name"`
	Nullable         bool   `json:"nullable,omitempty"`
	PartitionIndex   int    `json:"partition_index,omitempty"`
	Position         int    `json:"position"`
	TypeIntervalType string `json:"type_interval_type,omitempty"`
	TypeJson         string `json:"type_json,omitempty"`
	TypeName         string `json:"type_name"`
	TypePrecision    int    `json:"type_precision,omitempty"`
	TypeScale        int    `json:"type_scale,omitempty"`
	TypeText         string `json:"type_text"`
}

type ResourceTable struct {
	CatalogName           string                `json:"catalog_name"`
	Comment               string                `json:"comment,omitempty"`
	DataSourceFormat      string                `json:"data_source_format"`
	Id                    string                `json:"id,omitempty"`
	Name                  string                `json:"name"`
	Owner                 string                `json:"owner,omitempty"`
	Properties            map[string]string     `json:"properties,omitempty"`
	SchemaName            string                `json:"schema_name"`
	StorageCredentialName string                `json:"storage_credential_name,omitempty"`
	StorageLocation       string                `json:"storage_location,omitempty"`
	TableType             string                `json:"table_type"`
	ViewDefinition        string                `json:"view_definition,omitempty"`
	Column                []ResourceTableColumn `json:"column,omitempty"`
}
