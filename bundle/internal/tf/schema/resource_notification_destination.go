// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceNotificationDestinationConfigEmail struct {
	Addresses []string `json:"addresses,omitempty"`
}

type ResourceNotificationDestinationConfigGenericWebhook struct {
	Password    string `json:"password,omitempty"`
	PasswordSet bool   `json:"password_set,omitempty"`
	Url         string `json:"url,omitempty"`
	UrlSet      bool   `json:"url_set,omitempty"`
	Username    string `json:"username,omitempty"`
	UsernameSet bool   `json:"username_set,omitempty"`
}

type ResourceNotificationDestinationConfigMicrosoftTeams struct {
	AppId         string `json:"app_id,omitempty"`
	AppIdSet      bool   `json:"app_id_set,omitempty"`
	AuthSecret    string `json:"auth_secret,omitempty"`
	AuthSecretSet bool   `json:"auth_secret_set,omitempty"`
	ChannelUrl    string `json:"channel_url,omitempty"`
	ChannelUrlSet bool   `json:"channel_url_set,omitempty"`
	TenantId      string `json:"tenant_id,omitempty"`
	TenantIdSet   bool   `json:"tenant_id_set,omitempty"`
	Url           string `json:"url,omitempty"`
	UrlSet        bool   `json:"url_set,omitempty"`
}

type ResourceNotificationDestinationConfigPagerduty struct {
	IntegrationKey    string `json:"integration_key,omitempty"`
	IntegrationKeySet bool   `json:"integration_key_set,omitempty"`
}

type ResourceNotificationDestinationConfigSlack struct {
	ChannelId     string `json:"channel_id,omitempty"`
	ChannelIdSet  bool   `json:"channel_id_set,omitempty"`
	OauthToken    string `json:"oauth_token,omitempty"`
	OauthTokenSet bool   `json:"oauth_token_set,omitempty"`
	Url           string `json:"url,omitempty"`
	UrlSet        bool   `json:"url_set,omitempty"`
}

type ResourceNotificationDestinationConfig struct {
	Email          *ResourceNotificationDestinationConfigEmail          `json:"email,omitempty"`
	GenericWebhook *ResourceNotificationDestinationConfigGenericWebhook `json:"generic_webhook,omitempty"`
	MicrosoftTeams *ResourceNotificationDestinationConfigMicrosoftTeams `json:"microsoft_teams,omitempty"`
	Pagerduty      *ResourceNotificationDestinationConfigPagerduty      `json:"pagerduty,omitempty"`
	Slack          *ResourceNotificationDestinationConfigSlack          `json:"slack,omitempty"`
}

type ResourceNotificationDestinationProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceNotificationDestination struct {
	DestinationType string                                         `json:"destination_type,omitempty"`
	DisplayName     string                                         `json:"display_name"`
	Id              string                                         `json:"id,omitempty"`
	Config          *ResourceNotificationDestinationConfig         `json:"config,omitempty"`
	ProviderConfig  *ResourceNotificationDestinationProviderConfig `json:"provider_config,omitempty"`
}
