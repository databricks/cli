package resources

// ModifiedStatus is an enum of the possible statuses of a resource from the local perspective.
// CREATED - new resources that have been added to the local bundle configuration and don't yet exist remotely.
// DELETED - existing resources that have been removed from the local bundle but still exist remotely.
// UPDATED - existing resources that have been modified.
// An empty status means that the resource is unchanged.
// We use these statuses to build git-status-like UI of the resources in the Databricks VSCode extension.
type ModifiedStatus = string

const (
	ModifiedStatusCreated ModifiedStatus = "created"
	ModifiedStatusDeleted ModifiedStatus = "deleted"
)
