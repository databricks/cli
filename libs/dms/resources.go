package dms

// Resource is what DMS holds for one resource of a deployment, as of the last operation
// that recorded it.
type Resource struct {
	// Key is the bundle state key, as everything outside this package spells it.
	Key string
	ID  string

	// State is the state the last operation recorded, as the opaque string the service
	// stores, and empty when no operation recorded one.
	State string
}
