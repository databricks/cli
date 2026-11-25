package resources

// Approval declares the data-loss warnings of a resource type. "bundle deploy" asks for
// consent only when it deletes or recreates a resource whose type has a Deploy message,
// and prints that message above those resources. "bundle destroy" always asks, and first
// prints the Destroy message above the resources of this type that it deletes.
type Approval struct {
	// Order places this type's messages among those of other types: lower prints first.
	// It is explicit because the order in which the messages print predates the canonical
	// order of resource types. Every type with a message has a distinct Order.
	Order int

	// Deploy is the message for the deletions and recreations of "bundle deploy". Empty
	// means that they need no consent.
	Deploy string

	// DeploySkipChildren lists only top-level resources under Deploy. Deleting or
	// recreating a child resource (such as grants) still requires consent.
	DeploySkipChildren bool

	// Destroy is the message for the deletions of "bundle destroy". Empty means no message.
	Destroy string

	// DestroyFromState means that the destroy message depends on the deployment state of
	// each resource, so a hook registered for the type in bundle/phases picks it, and
	// Destroy is empty. These messages print after the Destroy messages of all types.
	DestroyFromState bool
}

// ApprovalDeclarer is implemented by every resource type.
type ApprovalDeclarer interface {
	// Approval returns the type's data-loss warnings. The zero Approval declares that
	// deleting or recreating a resource of the type needs no warning.
	Approval() Approval
}
