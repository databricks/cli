// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceExternalLocationEncryptionDetailsSseEncryptionDetails struct {
	Algorithm    string `json:"algorithm,omitempty"`
	AwsKmsKeyArn string `json:"aws_kms_key_arn,omitempty"`
}

type ResourceExternalLocationEncryptionDetails struct {
	SseEncryptionDetails *ResourceExternalLocationEncryptionDetailsSseEncryptionDetails `json:"sse_encryption_details,omitempty"`
}

type ResourceExternalLocation struct {
	AccessPoint       string                                     `json:"access_point,omitempty"`
	BrowseOnly        bool                                       `json:"browse_only,omitempty"`
	Comment           string                                     `json:"comment,omitempty"`
	CreatedAt         int                                        `json:"created_at,omitempty"`
	CreatedBy         string                                     `json:"created_by,omitempty"`
	CredentialId      string                                     `json:"credential_id,omitempty"`
	CredentialName    string                                     `json:"credential_name"`
	Fallback          bool                                       `json:"fallback,omitempty"`
	ForceDestroy      bool                                       `json:"force_destroy,omitempty"`
	ForceUpdate       bool                                       `json:"force_update,omitempty"`
	Id                string                                     `json:"id,omitempty"`
	IsolationMode     string                                     `json:"isolation_mode,omitempty"`
	MetastoreId       string                                     `json:"metastore_id,omitempty"`
	Name              string                                     `json:"name"`
	Owner             string                                     `json:"owner,omitempty"`
	ReadOnly          bool                                       `json:"read_only,omitempty"`
	SkipValidation    bool                                       `json:"skip_validation,omitempty"`
	UpdatedAt         int                                        `json:"updated_at,omitempty"`
	UpdatedBy         string                                     `json:"updated_by,omitempty"`
	Url               string                                     `json:"url"`
	EncryptionDetails *ResourceExternalLocationEncryptionDetails `json:"encryption_details,omitempty"`
}
