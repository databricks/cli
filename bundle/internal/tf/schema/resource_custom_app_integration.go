// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceCustomAppIntegrationTokenAccessPolicy struct {
	AbsoluteSessionLifetimeInMinutes int  `json:"absolute_session_lifetime_in_minutes,omitempty"`
	AccessTokenTtlInMinutes          int  `json:"access_token_ttl_in_minutes,omitempty"`
	EnableSingleUseRefreshTokens     bool `json:"enable_single_use_refresh_tokens,omitempty"`
	RefreshTokenTtlInMinutes         int  `json:"refresh_token_ttl_in_minutes,omitempty"`
}

type ResourceCustomAppIntegration struct {
	ClientId             string                                         `json:"client_id,omitempty"`
	ClientSecret         string                                         `json:"client_secret,omitempty"`
	Confidential         bool                                           `json:"confidential,omitempty"`
	CreateTime           string                                         `json:"create_time,omitempty"`
	CreatedBy            int                                            `json:"created_by,omitempty"`
	CreatorUsername      string                                         `json:"creator_username,omitempty"`
	Id                   string                                         `json:"id,omitempty"`
	IntegrationId        string                                         `json:"integration_id,omitempty"`
	Name                 string                                         `json:"name,omitempty"`
	RedirectUrls         []string                                       `json:"redirect_urls,omitempty"`
	Scopes               []string                                       `json:"scopes,omitempty"`
	UserAuthorizedScopes []string                                       `json:"user_authorized_scopes,omitempty"`
	TokenAccessPolicy    *ResourceCustomAppIntegrationTokenAccessPolicy `json:"token_access_policy,omitempty"`
}
