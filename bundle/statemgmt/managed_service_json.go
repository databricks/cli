package statemgmt

// ManagedServiceFileName is the workspace state-directory file that records the
// deployment metadata service (DMS) deployment_id for the bundle. The file
// pins this bundle to a server-side deployment record across CLI invocations.
// It is created by the lock package after the first CreateDeployment succeeds
// and is read by both the lock package (when re-acquiring the lock) and the
// statemgmt package (when loading resource state from DMS).
//
// resources.json continues to be written by the deploy path so that operators
// who turn DATABRICKS_BUNDLE_MANAGED_STATE off again still have a usable local
// state file. The fallback path is intentional, not accidental.
const ManagedServiceFileName = "managed_service.json"

// ManagedServiceJSON is the on-disk shape of ManagedServiceFileName.
type ManagedServiceJSON struct {
	DeploymentID string `json:"deployment_id"`
}
