// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceRfaAccessRequestDestinationsDestinationSourceSecurable struct {
	FullName      string `json:"full_name,omitempty"`
	ProviderShare string `json:"provider_share,omitempty"`
	Type          string `json:"type,omitempty"`
}

type DataSourceRfaAccessRequestDestinationsDestinations struct {
	DestinationId      string `json:"destination_id,omitempty"`
	DestinationType    string `json:"destination_type,omitempty"`
	SpecialDestination string `json:"special_destination,omitempty"`
}

type DataSourceRfaAccessRequestDestinationsSecurable struct {
	FullName      string `json:"full_name,omitempty"`
	ProviderShare string `json:"provider_share,omitempty"`
	Type          string `json:"type,omitempty"`
}

type DataSourceRfaAccessRequestDestinations struct {
	AreAnyDestinationsHidden   bool                                                              `json:"are_any_destinations_hidden,omitempty"`
	DestinationSourceSecurable *DataSourceRfaAccessRequestDestinationsDestinationSourceSecurable `json:"destination_source_securable,omitempty"`
	Destinations               []DataSourceRfaAccessRequestDestinationsDestinations              `json:"destinations,omitempty"`
	FullName                   string                                                            `json:"full_name"`
	Securable                  *DataSourceRfaAccessRequestDestinationsSecurable                  `json:"securable,omitempty"`
	SecurableType              string                                                            `json:"securable_type"`
}
