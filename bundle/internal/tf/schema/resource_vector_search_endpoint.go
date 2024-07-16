// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceVectorSearchEndpoint struct {
	CreationTimestamp    int    `json:"creation_timestamp,omitempty"`
	Creator              string `json:"creator,omitempty"`
	EndpointId           string `json:"endpoint_id,omitempty"`
	EndpointStatus       []any  `json:"endpoint_status,omitempty"`
	EndpointType         string `json:"endpoint_type"`
	Id                   string `json:"id,omitempty"`
	LastUpdatedTimestamp int    `json:"last_updated_timestamp,omitempty"`
	LastUpdatedUser      string `json:"last_updated_user,omitempty"`
	Name                 string `json:"name"`
	NumIndexes           int    `json:"num_indexes,omitempty"`
}
