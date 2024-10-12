// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceExternalLocationExternalLocationInfoEncryptionDetailsSseEncryptionDetails struct {
	Algorithm    string `json:"algorithm,omitempty"`
	AwsKmsKeyArn string `json:"aws_kms_key_arn,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoEncryptionDetails struct {
	SseEncryptionDetails *DataSourceExternalLocationExternalLocationInfoEncryptionDetailsSseEncryptionDetails `json:"sse_encryption_details,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfo struct {
	AccessPoint       string                                                           `json:"access_point,omitempty"`
	BrowseOnly        bool                                                             `json:"browse_only,omitempty"`
	Comment           string                                                           `json:"comment,omitempty"`
	CreatedAt         int                                                              `json:"created_at,omitempty"`
	CreatedBy         string                                                           `json:"created_by,omitempty"`
	CredentialId      string                                                           `json:"credential_id,omitempty"`
	CredentialName    string                                                           `json:"credential_name,omitempty"`
	Fallback          bool                                                             `json:"fallback,omitempty"`
	IsolationMode     string                                                           `json:"isolation_mode,omitempty"`
	MetastoreId       string                                                           `json:"metastore_id,omitempty"`
	Name              string                                                           `json:"name,omitempty"`
	Owner             string                                                           `json:"owner,omitempty"`
	ReadOnly          bool                                                             `json:"read_only,omitempty"`
	UpdatedAt         int                                                              `json:"updated_at,omitempty"`
	UpdatedBy         string                                                           `json:"updated_by,omitempty"`
	Url               string                                                           `json:"url,omitempty"`
	EncryptionDetails *DataSourceExternalLocationExternalLocationInfoEncryptionDetails `json:"encryption_details,omitempty"`
}

type DataSourceExternalLocation struct {
	Id                   string                                          `json:"id,omitempty"`
	Name                 string                                          `json:"name"`
	ExternalLocationInfo *DataSourceExternalLocationExternalLocationInfo `json:"external_location_info,omitempty"`
}
