// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRfaAccessRequestDestinationsDestinations struct {
	DestinationId      string `json:"destination_id,omitempty"`
	DestinationType    string `json:"destination_type,omitempty"`
	SpecialDestination string `json:"special_destination,omitempty"`
}

type ResourceRfaAccessRequestDestinationsSecurable struct {
	FullName      string `json:"full_name,omitempty"`
	ProviderShare string `json:"provider_share,omitempty"`
	Type          string `json:"type,omitempty"`
}

type ResourceRfaAccessRequestDestinations struct {
	AreAnyDestinationsHidden bool                                               `json:"are_any_destinations_hidden,omitempty"`
	Destinations             []ResourceRfaAccessRequestDestinationsDestinations `json:"destinations,omitempty"`
	Securable                *ResourceRfaAccessRequestDestinationsSecurable     `json:"securable,omitempty"`
}
