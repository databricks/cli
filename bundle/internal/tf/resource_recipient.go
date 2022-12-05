// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceRecipientIpAccessList struct {
	AllowedIpAddresses []string `json:"allowed_ip_addresses"`
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
	AuthenticationType             string                         `json:"authentication_type"`
	Comment                        string                         `json:"comment,omitempty"`
	DataRecipientGlobalMetastoreId string                         `json:"data_recipient_global_metastore_id,omitempty"`
	Id                             string                         `json:"id,omitempty"`
	Name                           string                         `json:"name"`
	SharingCode                    string                         `json:"sharing_code,omitempty"`
	IpAccessList                   *ResourceRecipientIpAccessList `json:"ip_access_list,omitempty"`
	Tokens                         []ResourceRecipientTokens      `json:"tokens,omitempty"`
}
