package resources

// GrantsDeclarer is implemented by every resource type that has a grants field.
type GrantsDeclarer interface {
	// GrantsSupport is called on a zero value and must not read the receiver.
	GrantsSupport() GrantsSupport
}

// GrantsSupport describes how the grants of a resource type are deployed.
type GrantsSupport struct {
	// SecurableType is the Unity Catalog securable type in grants API requests, for example "schema".
	SecurableType string
}
