// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRequestForAccessDestinations struct {
	DestinationId      string `json:"destination_id,omitempty"`
	DestinationType    string `json:"destination_type,omitempty"`
	SpecialDestination string `json:"special_destination,omitempty"`
}

type ResourceRequestForAccessSecurable struct {
	FullName      string `json:"full_name,omitempty"`
	ProviderShare string `json:"provider_share,omitempty"`
	Type          string `json:"type,omitempty"`
}

type ResourceRequestForAccess struct {
	AreAnyDestinationsHidden bool                                   `json:"are_any_destinations_hidden,omitempty"`
	Destinations             []ResourceRequestForAccessDestinations `json:"destinations,omitempty"`
	Securable                *ResourceRequestForAccessSecurable     `json:"securable,omitempty"`
}
