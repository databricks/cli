// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCleanRoomsCleanRoomOutputCatalog struct {
	CatalogName string `json:"catalog_name,omitempty"`
	Status      string `json:"status,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoCollaborators struct {
	CollaboratorAlias          string `json:"collaborator_alias"`
	DisplayName                string `json:"display_name,omitempty"`
	GlobalMetastoreId          string `json:"global_metastore_id,omitempty"`
	InviteRecipientEmail       string `json:"invite_recipient_email,omitempty"`
	InviteRecipientWorkspaceId int    `json:"invite_recipient_workspace_id,omitempty"`
	OrganizationName           string `json:"organization_name,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoComplianceSecurityProfile struct {
	ComplianceStandards []string `json:"compliance_standards,omitempty"`
	IsEnabled           bool     `json:"is_enabled,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoCreator struct {
	CollaboratorAlias          string `json:"collaborator_alias"`
	DisplayName                string `json:"display_name,omitempty"`
	GlobalMetastoreId          string `json:"global_metastore_id,omitempty"`
	InviteRecipientEmail       string `json:"invite_recipient_email,omitempty"`
	InviteRecipientWorkspaceId int    `json:"invite_recipient_workspace_id,omitempty"`
	OrganizationName           string `json:"organization_name,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedInternetDestinations struct {
	Destination string `json:"destination,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Type        string `json:"type,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedStorageDestinations struct {
	AllowedPaths        []string `json:"allowed_paths,omitempty"`
	AzureContainer      string   `json:"azure_container,omitempty"`
	AzureDnsZone        string   `json:"azure_dns_zone,omitempty"`
	AzureStorageAccount string   `json:"azure_storage_account,omitempty"`
	AzureStorageService string   `json:"azure_storage_service,omitempty"`
	BucketName          string   `json:"bucket_name,omitempty"`
	Region              string   `json:"region,omitempty"`
	Type                string   `json:"type,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessLogOnlyMode struct {
	LogOnlyModeType string   `json:"log_only_mode_type,omitempty"`
	Workloads       []string `json:"workloads,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccess struct {
	AllowedInternetDestinations []DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedInternetDestinations `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations  []DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedStorageDestinations  `json:"allowed_storage_destinations,omitempty"`
	LogOnlyMode                 *DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessLogOnlyMode                  `json:"log_only_mode,omitempty"`
	RestrictionMode             string                                                                                                        `json:"restriction_mode,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicy struct {
	InternetAccess *DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccess `json:"internet_access,omitempty"`
}

type DataSourceCleanRoomsCleanRoomRemoteDetailedInfo struct {
	CentralCleanRoomId        string                                                                    `json:"central_clean_room_id,omitempty"`
	CloudVendor               string                                                                    `json:"cloud_vendor,omitempty"`
	Collaborators             []DataSourceCleanRoomsCleanRoomRemoteDetailedInfoCollaborators            `json:"collaborators,omitempty"`
	ComplianceSecurityProfile *DataSourceCleanRoomsCleanRoomRemoteDetailedInfoComplianceSecurityProfile `json:"compliance_security_profile,omitempty"`
	Creator                   *DataSourceCleanRoomsCleanRoomRemoteDetailedInfoCreator                   `json:"creator,omitempty"`
	EgressNetworkPolicy       *DataSourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicy       `json:"egress_network_policy,omitempty"`
	Region                    string                                                                    `json:"region,omitempty"`
}

type DataSourceCleanRoomsCleanRoom struct {
	AccessRestricted       string                                           `json:"access_restricted,omitempty"`
	Comment                string                                           `json:"comment,omitempty"`
	CreatedAt              int                                              `json:"created_at,omitempty"`
	LocalCollaboratorAlias string                                           `json:"local_collaborator_alias,omitempty"`
	Name                   string                                           `json:"name,omitempty"`
	OutputCatalog          *DataSourceCleanRoomsCleanRoomOutputCatalog      `json:"output_catalog,omitempty"`
	Owner                  string                                           `json:"owner,omitempty"`
	RemoteDetailedInfo     *DataSourceCleanRoomsCleanRoomRemoteDetailedInfo `json:"remote_detailed_info,omitempty"`
	Status                 string                                           `json:"status,omitempty"`
	UpdatedAt              int                                              `json:"updated_at,omitempty"`
}
