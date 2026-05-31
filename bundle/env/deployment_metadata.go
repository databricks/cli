package env

// ManagedStateVariable opts a bundle into the deployment metadata service
// (DMS) for locking and resource-state management. Defaults to the historical
// filesystem-based behavior when unset.
//
// The variable is treated as a boolean and accepts the usual spellings:
// "true"/"false", "1"/"0", "yes"/"no", "on"/"off" (case-insensitive).
const ManagedStateVariable = "DATABRICKS_BUNDLE_MANAGED_STATE"
