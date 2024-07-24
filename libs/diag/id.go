package diag

type ID string

// For select diagnostic messages we use IDs to identify them
// for support or tooling purposes.
// It is a non-goal to have an exhaustive list of IDs.
const (
	// We have many subtly different permission errors.
	// These are numbered for easy reference and tooling support.
	PathPermissionDenied        ID = "EPERM1"
	ResourcePermissionDenied    ID = "EPERM2"
	CannotChangePathPermissions ID = "EPERM3"
	RunAsDenied                 ID = "EPERM4"
	PermissionNotIncluded       ID = "EPERM5"
)
