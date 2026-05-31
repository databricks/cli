package env

// ManagedStateVariable opts a bundle into the deployment metadata service
// (DMS) for locking and resource-state management. Only the exact value
// "true" enables managed state; any other value is treated as unset.
const ManagedStateVariable = "DATABRICKS_BUNDLE_MANAGED_STATE"
