// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRecipientIpAccessList struct {
	AllowedIpAddresses []string `json:"allowed_ip_addresses,omitempty"`
}

type ResourceRecipientPropertiesKvpairs struct {
	Properties map[string]string `json:"properties"`
}

type ResourceRecipientTokens struct {
	ActivationUrl  string `json:"activation_url,omitempty"`
	CreatedAt      int    `json:"created_at,omitempty"`
	CreatedBy      string `json:"created_by,omitempty"`
	ExpirationTime int    `json:"expiration_time,omitempty"`
	Id             string `json:"id,omitempty"`
	UpdatedAt      int    `json:"updated_at,omitempty"`
	UpdatedBy      string `json:"updated_by,omitempty"`
}

type ResourceRecipient struct {
	Activated                      bool                                `json:"activated,omitempty"`
	ActivationUrl                  string                              `json:"activation_url,omitempty"`
	AuthenticationType             string                              `json:"authentication_type"`
	Cloud                          string                              `json:"cloud,omitempty"`
	Comment                        string                              `json:"comment,omitempty"`
	CreatedAt                      int                                 `json:"created_at,omitempty"`
	CreatedBy                      string                              `json:"created_by,omitempty"`
	DataRecipientGlobalMetastoreId string                              `json:"data_recipient_global_metastore_id,omitempty"`
	ExpirationTime                 int                                 `json:"expiration_time,omitempty"`
	Id                             string                              `json:"id,omitempty"`
	MetastoreId                    string                              `json:"metastore_id,omitempty"`
	Name                           string                              `json:"name"`
	Owner                          string                              `json:"owner,omitempty"`
	Region                         string                              `json:"region,omitempty"`
	SharingCode                    string                              `json:"sharing_code,omitempty"`
	UpdatedAt                      int                                 `json:"updated_at,omitempty"`
	UpdatedBy                      string                              `json:"updated_by,omitempty"`
	IpAccessList                   *ResourceRecipientIpAccessList      `json:"ip_access_list,omitempty"`
	PropertiesKvpairs              *ResourceRecipientPropertiesKvpairs `json:"properties_kvpairs,omitempty"`
	Tokens                         []ResourceRecipientTokens           `json:"tokens,omitempty"`
}
