// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceCleanRoomsCleanRoomOutputCatalog struct {
	CatalogName string `json:"catalog_name,omitempty"`
	Status      string `json:"status,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoCollaborators struct {
	CollaboratorAlias          string `json:"collaborator_alias"`
	DisplayName                string `json:"display_name,omitempty"`
	GlobalMetastoreId          string `json:"global_metastore_id,omitempty"`
	InviteRecipientEmail       string `json:"invite_recipient_email,omitempty"`
	InviteRecipientWorkspaceId int    `json:"invite_recipient_workspace_id,omitempty"`
	OrganizationName           string `json:"organization_name,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoComplianceSecurityProfile struct {
	ComplianceStandards []string `json:"compliance_standards,omitempty"`
	IsEnabled           bool     `json:"is_enabled,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoCreator struct {
	CollaboratorAlias          string `json:"collaborator_alias"`
	DisplayName                string `json:"display_name,omitempty"`
	GlobalMetastoreId          string `json:"global_metastore_id,omitempty"`
	InviteRecipientEmail       string `json:"invite_recipient_email,omitempty"`
	InviteRecipientWorkspaceId int    `json:"invite_recipient_workspace_id,omitempty"`
	OrganizationName           string `json:"organization_name,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedInternetDestinations struct {
	Destination string `json:"destination,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Type        string `json:"type,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedStorageDestinations struct {
	AllowedPaths        []string `json:"allowed_paths,omitempty"`
	AzureContainer      string   `json:"azure_container,omitempty"`
	AzureDnsZone        string   `json:"azure_dns_zone,omitempty"`
	AzureStorageAccount string   `json:"azure_storage_account,omitempty"`
	AzureStorageService string   `json:"azure_storage_service,omitempty"`
	BucketName          string   `json:"bucket_name,omitempty"`
	Region              string   `json:"region,omitempty"`
	Type                string   `json:"type,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessLogOnlyMode struct {
	LogOnlyModeType string   `json:"log_only_mode_type,omitempty"`
	Workloads       []string `json:"workloads,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccess struct {
	AllowedInternetDestinations []ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedInternetDestinations `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations  []ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessAllowedStorageDestinations  `json:"allowed_storage_destinations,omitempty"`
	LogOnlyMode                 *ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccessLogOnlyMode                  `json:"log_only_mode,omitempty"`
	RestrictionMode             string                                                                                                      `json:"restriction_mode,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicy struct {
	InternetAccess *ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicyInternetAccess `json:"internet_access,omitempty"`
}

type ResourceCleanRoomsCleanRoomRemoteDetailedInfo struct {
	CentralCleanRoomId        string                                                                  `json:"central_clean_room_id,omitempty"`
	CloudVendor               string                                                                  `json:"cloud_vendor,omitempty"`
	Collaborators             []ResourceCleanRoomsCleanRoomRemoteDetailedInfoCollaborators            `json:"collaborators,omitempty"`
	ComplianceSecurityProfile *ResourceCleanRoomsCleanRoomRemoteDetailedInfoComplianceSecurityProfile `json:"compliance_security_profile,omitempty"`
	Creator                   *ResourceCleanRoomsCleanRoomRemoteDetailedInfoCreator                   `json:"creator,omitempty"`
	EgressNetworkPolicy       *ResourceCleanRoomsCleanRoomRemoteDetailedInfoEgressNetworkPolicy       `json:"egress_network_policy,omitempty"`
	Region                    string                                                                  `json:"region,omitempty"`
}

type ResourceCleanRoomsCleanRoom struct {
	AccessRestricted       string                                         `json:"access_restricted,omitempty"`
	Comment                string                                         `json:"comment,omitempty"`
	CreatedAt              int                                            `json:"created_at,omitempty"`
	LocalCollaboratorAlias string                                         `json:"local_collaborator_alias,omitempty"`
	Name                   string                                         `json:"name,omitempty"`
	OutputCatalog          *ResourceCleanRoomsCleanRoomOutputCatalog      `json:"output_catalog,omitempty"`
	Owner                  string                                         `json:"owner,omitempty"`
	RemoteDetailedInfo     *ResourceCleanRoomsCleanRoomRemoteDetailedInfo `json:"remote_detailed_info,omitempty"`
	Status                 string                                         `json:"status,omitempty"`
	UpdatedAt              int                                            `json:"updated_at,omitempty"`
}
