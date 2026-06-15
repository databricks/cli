package statemgmt

// ManagedServiceFileName is the filename for ManagedServiceJSON in the workspace
// state directory.
const ManagedServiceFileName = "managed_service.json"

// ManagedServiceJSON holds DMS-specific bookkeeping (e.g. the deployment_id
// that ties this bundle to a server-side deployment record). It lives next to
// resources.json in the workspace state directory. resources.json is kept
// around and maintained alongside this file so users have a backward path if
// they hit issues with the DMS-backed deployment flow.
type ManagedServiceJSON struct {
	DeploymentID string `json:"deployment_id"`
}
