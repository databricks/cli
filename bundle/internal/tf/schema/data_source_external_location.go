// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceExternalLocationExternalLocationInfoEncryptionDetailsSseEncryptionDetails struct {
	Algorithm    string `json:"algorithm,omitempty"`
	AwsKmsKeyArn string `json:"aws_kms_key_arn,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoEncryptionDetails struct {
	SseEncryptionDetails *DataSourceExternalLocationExternalLocationInfoEncryptionDetailsSseEncryptionDetails `json:"sse_encryption_details,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoFileEventQueueManagedAqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url,omitempty"`
	ResourceGroup     string `json:"resource_group,omitempty"`
	SubscriptionId    string `json:"subscription_id,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoFileEventQueueManagedPubsub struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	SubscriptionName  string `json:"subscription_name,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoFileEventQueueManagedSqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoFileEventQueueProvidedAqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url,omitempty"`
	ResourceGroup     string `json:"resource_group,omitempty"`
	SubscriptionId    string `json:"subscription_id,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoFileEventQueueProvidedPubsub struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	SubscriptionName  string `json:"subscription_name,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoFileEventQueueProvidedSqs struct {
	ManagedResourceId string `json:"managed_resource_id,omitempty"`
	QueueUrl          string `json:"queue_url,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfoFileEventQueue struct {
	ManagedAqs     *DataSourceExternalLocationExternalLocationInfoFileEventQueueManagedAqs     `json:"managed_aqs,omitempty"`
	ManagedPubsub  *DataSourceExternalLocationExternalLocationInfoFileEventQueueManagedPubsub  `json:"managed_pubsub,omitempty"`
	ManagedSqs     *DataSourceExternalLocationExternalLocationInfoFileEventQueueManagedSqs     `json:"managed_sqs,omitempty"`
	ProvidedAqs    *DataSourceExternalLocationExternalLocationInfoFileEventQueueProvidedAqs    `json:"provided_aqs,omitempty"`
	ProvidedPubsub *DataSourceExternalLocationExternalLocationInfoFileEventQueueProvidedPubsub `json:"provided_pubsub,omitempty"`
	ProvidedSqs    *DataSourceExternalLocationExternalLocationInfoFileEventQueueProvidedSqs    `json:"provided_sqs,omitempty"`
}

type DataSourceExternalLocationExternalLocationInfo struct {
	BrowseOnly        bool                                                             `json:"browse_only,omitempty"`
	Comment           string                                                           `json:"comment,omitempty"`
	CreatedAt         int                                                              `json:"created_at,omitempty"`
	CreatedBy         string                                                           `json:"created_by,omitempty"`
	CredentialId      string                                                           `json:"credential_id,omitempty"`
	CredentialName    string                                                           `json:"credential_name,omitempty"`
	EnableFileEvents  bool                                                             `json:"enable_file_events,omitempty"`
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
	FileEventQueue    *DataSourceExternalLocationExternalLocationInfoFileEventQueue    `json:"file_event_queue,omitempty"`
}

type DataSourceExternalLocation struct {
	Id                   string                                          `json:"id,omitempty"`
	Name                 string                                          `json:"name"`
	ExternalLocationInfo *DataSourceExternalLocationExternalLocationInfo `json:"external_location_info,omitempty"`
}
