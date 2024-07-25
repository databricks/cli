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
	Comment           string                                     `json:"comment,omitempty"`
	CredentialName    string                                     `json:"credential_name"`
	ForceDestroy      bool                                       `json:"force_destroy,omitempty"`
	ForceUpdate       bool                                       `json:"force_update,omitempty"`
	Id                string                                     `json:"id,omitempty"`
	MetastoreId       string                                     `json:"metastore_id,omitempty"`
	Name              string                                     `json:"name"`
	Owner             string                                     `json:"owner,omitempty"`
	ReadOnly          bool                                       `json:"read_only,omitempty"`
	SkipValidation    bool                                       `json:"skip_validation,omitempty"`
	Url               string                                     `json:"url"`
	EncryptionDetails *ResourceExternalLocationEncryptionDetails `json:"encryption_details,omitempty"`
}
