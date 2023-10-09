package deploy

import "github.com/databricks/cli/bundle/config"

var LatestMetadataVersion = 1

var MetadataTag = "metadata"

// Metadata about the bundle deployment. This is the interface Databricks services
// rely on to integrate with bundles when they need additional information about
// a bundle deployment.
//
// After deploy, a file containing the metadata can be found at the workspace file system.
type Metadata struct {
	Version int `json:"version"`

	Config config.Root `json:"config"`
}
