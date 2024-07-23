// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceVolumeVolumeInfoEncryptionDetailsSseEncryptionDetails struct {
	Algorithm    string `json:"algorithm,omitempty"`
	AwsKmsKeyArn string `json:"aws_kms_key_arn,omitempty"`
}

type DataSourceVolumeVolumeInfoEncryptionDetails struct {
	SseEncryptionDetails *DataSourceVolumeVolumeInfoEncryptionDetailsSseEncryptionDetails `json:"sse_encryption_details,omitempty"`
}

type DataSourceVolumeVolumeInfo struct {
	AccessPoint       string                                       `json:"access_point,omitempty"`
	BrowseOnly        bool                                         `json:"browse_only,omitempty"`
	CatalogName       string                                       `json:"catalog_name,omitempty"`
	Comment           string                                       `json:"comment,omitempty"`
	CreatedAt         int                                          `json:"created_at,omitempty"`
	CreatedBy         string                                       `json:"created_by,omitempty"`
	FullName          string                                       `json:"full_name,omitempty"`
	MetastoreId       string                                       `json:"metastore_id,omitempty"`
	Name              string                                       `json:"name,omitempty"`
	Owner             string                                       `json:"owner,omitempty"`
	SchemaName        string                                       `json:"schema_name,omitempty"`
	StorageLocation   string                                       `json:"storage_location,omitempty"`
	UpdatedAt         int                                          `json:"updated_at,omitempty"`
	UpdatedBy         string                                       `json:"updated_by,omitempty"`
	VolumeId          string                                       `json:"volume_id,omitempty"`
	VolumeType        string                                       `json:"volume_type,omitempty"`
	EncryptionDetails *DataSourceVolumeVolumeInfoEncryptionDetails `json:"encryption_details,omitempty"`
}

type DataSourceVolume struct {
	Id         string                      `json:"id,omitempty"`
	Name       string                      `json:"name"`
	VolumeInfo *DataSourceVolumeVolumeInfo `json:"volume_info,omitempty"`
}
