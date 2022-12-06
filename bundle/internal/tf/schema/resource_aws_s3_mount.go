// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAwsS3Mount struct {
	ClusterId       string `json:"cluster_id,omitempty"`
	Id              string `json:"id,omitempty"`
	InstanceProfile string `json:"instance_profile,omitempty"`
	MountName       string `json:"mount_name"`
	S3BucketName    string `json:"s3_bucket_name"`
	Source          string `json:"source,omitempty"`
}
