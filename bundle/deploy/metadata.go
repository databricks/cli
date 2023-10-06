package deploy

import "github.com/databricks/cli/bundle/config"

var LatestMetadataVersion = 1

var MetadataTag = "metadata"

// TODO: refine these comments
// Metadata is a select list of fields from the bundle config that is written
// to the WSFS on bundle deploy. Databricks services can read this file to gather
// more information about a particular bundle deployment.
//
// Post deploy, the metadata file can be found at ${workspace.state_path}/deploy-metadata.json
type Metadata struct {
	Version int `json:"version"`

	Config config.Root `json:"config"`
}
