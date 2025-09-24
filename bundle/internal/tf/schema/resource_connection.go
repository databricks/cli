// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceConnection struct {
	Comment          string            `json:"comment,omitempty"`
	ConnectionId     string            `json:"connection_id,omitempty"`
	ConnectionType   string            `json:"connection_type,omitempty"`
	CreatedAt        int               `json:"created_at,omitempty"`
	CreatedBy        string            `json:"created_by,omitempty"`
	CredentialType   string            `json:"credential_type,omitempty"`
	FullName         string            `json:"full_name,omitempty"`
	Id               string            `json:"id,omitempty"`
	MetastoreId      string            `json:"metastore_id,omitempty"`
	Name             string            `json:"name,omitempty"`
	Options          map[string]string `json:"options,omitempty"`
	Owner            string            `json:"owner,omitempty"`
	Properties       map[string]string `json:"properties,omitempty"`
	ProvisioningInfo []any             `json:"provisioning_info,omitempty"`
	ReadOnly         bool              `json:"read_only,omitempty"`
	SecurableType    string            `json:"securable_type,omitempty"`
	UpdatedAt        int               `json:"updated_at,omitempty"`
	UpdatedBy        string            `json:"updated_by,omitempty"`
	Url              string            `json:"url,omitempty"`
}
