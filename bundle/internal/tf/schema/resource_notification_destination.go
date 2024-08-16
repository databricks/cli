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
	Url    string `json:"url,omitempty"`
	UrlSet bool   `json:"url_set,omitempty"`
}

type ResourceNotificationDestinationConfigPagerduty struct {
	IntegrationKey    string `json:"integration_key,omitempty"`
	IntegrationKeySet bool   `json:"integration_key_set,omitempty"`
}

type ResourceNotificationDestinationConfigSlack struct {
	Url    string `json:"url,omitempty"`
	UrlSet bool   `json:"url_set,omitempty"`
}

type ResourceNotificationDestinationConfig struct {
	Email          *ResourceNotificationDestinationConfigEmail          `json:"email,omitempty"`
	GenericWebhook *ResourceNotificationDestinationConfigGenericWebhook `json:"generic_webhook,omitempty"`
	MicrosoftTeams *ResourceNotificationDestinationConfigMicrosoftTeams `json:"microsoft_teams,omitempty"`
	Pagerduty      *ResourceNotificationDestinationConfigPagerduty      `json:"pagerduty,omitempty"`
	Slack          *ResourceNotificationDestinationConfigSlack          `json:"slack,omitempty"`
}

type ResourceNotificationDestination struct {
	DestinationType string                                 `json:"destination_type,omitempty"`
	DisplayName     string                                 `json:"display_name"`
	Id              string                                 `json:"id,omitempty"`
	Config          *ResourceNotificationDestinationConfig `json:"config,omitempty"`
}
