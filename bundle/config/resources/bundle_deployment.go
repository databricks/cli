package resources

// BundleDeploymentTracked is implemented by resource types whose settings have a
// "deployment" block in which the CLI records the bundle deployment that manages the
// resource. The CLI sets these values on every deploy, so the user must not set them.
type BundleDeploymentTracked interface {
	// BundleDeploymentIDs returns the configured deployment.deployment_id and
	// deployment.version_id. Unset values are empty.
	BundleDeploymentIDs() (deploymentID, versionID string)
}
