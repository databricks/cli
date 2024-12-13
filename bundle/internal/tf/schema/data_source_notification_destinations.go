// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceNotificationDestinationsNotificationDestinations struct {
	DestinationType string `json:"destination_type,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
	Id              string `json:"id,omitempty"`
}

type DataSourceNotificationDestinations struct {
	DisplayNameContains      string                                                       `json:"display_name_contains,omitempty"`
	NotificationDestinations []DataSourceNotificationDestinationsNotificationDestinations `json:"notification_destinations,omitempty"`
	Type                     string                                                       `json:"type,omitempty"`
}
