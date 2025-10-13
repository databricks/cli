// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceExternalLocationEncryptionDetailsSseEncryptionDetails struct {
	Algorithm    string `json:"algorithm,omitempty"`
	AwsKmsKeyArn string `json:"aws_kms_key_arn,omitempty"`
}

type ResourceExternalLocationEncryptionDetails struct {
	SseEncryptionDetails *ResourceExternalLocationEncryptionDetailsSseEncryptionDetails `json:"sse_encryption_details,omitempty"`
}

type ResourceExternalLocationFileEventQueueManagedAqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url,omitempty"`
	ResourceGroup     string `json:"resource_group"`
	SubscriptionId    string `json:"subscription_id"`
}

type ResourceExternalLocationFileEventQueueManagedPubsub struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	SubscriptionName  string `json:"subscription_name,omitempty"`
}

type ResourceExternalLocationFileEventQueueManagedSqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url,omitempty"`
}

type ResourceExternalLocationFileEventQueueProvidedAqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url"`
	ResourceGroup     string `json:"resource_group,omitempty"`
	SubscriptionId    string `json:"subscription_id,omitempty"`
}

type ResourceExternalLocationFileEventQueueProvidedPubsub struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	SubscriptionName  string `json:"subscription_name"`
}

type ResourceExternalLocationFileEventQueueProvidedSqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url"`
}

type ResourceExternalLocationFileEventQueue struct {
	ManagedAqs     *ResourceExternalLocationFileEventQueueManagedAqs     `json:"managed_aqs,omitempty"`
	ManagedPubsub  *ResourceExternalLocationFileEventQueueManagedPubsub  `json:"managed_pubsub,omitempty"`
	ManagedSqs     *ResourceExternalLocationFileEventQueueManagedSqs     `json:"managed_sqs,omitempty"`
	ProvidedAqs    *ResourceExternalLocationFileEventQueueProvidedAqs    `json:"provided_aqs,omitempty"`
	ProvidedPubsub *ResourceExternalLocationFileEventQueueProvidedPubsub `json:"provided_pubsub,omitempty"`
	ProvidedSqs    *ResourceExternalLocationFileEventQueueProvidedSqs    `json:"provided_sqs,omitempty"`
}

type ResourceExternalLocation struct {
	BrowseOnly        bool                                       `json:"browse_only,omitempty"`
	Comment           string                                     `json:"comment,omitempty"`
	CreatedAt         int                                        `json:"created_at,omitempty"`
	CreatedBy         string                                     `json:"created_by,omitempty"`
	CredentialId      string                                     `json:"credential_id,omitempty"`
	CredentialName    string                                     `json:"credential_name"`
	EnableFileEvents  bool                                       `json:"enable_file_events,omitempty"`
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
	FileEventQueue    *ResourceExternalLocationFileEventQueue    `json:"file_event_queue,omitempty"`
}
