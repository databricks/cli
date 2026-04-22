package config

import "github.com/databricks/cli/ucm/config/resources"

// Resources is the top-level container for every UC and cloud-underlay
// resource declared in ucm.yml. For M0 only a minimal UC-native subset is
// supported; cloud resources (S3/ADLS/GCS, IAM/MI/SA, KMS) land in M2.
type Resources struct {
	Catalogs           map[string]*resources.Catalog           `json:"catalogs,omitempty"`
	Schemas            map[string]*resources.Schema            `json:"schemas,omitempty"`
	Grants             map[string]*resources.Grant             `json:"grants,omitempty"`
	StorageCredentials map[string]*resources.StorageCredential `json:"storage_credentials,omitempty"`
	ExternalLocations  map[string]*resources.ExternalLocation  `json:"external_locations,omitempty"`
	TagValidationRules map[string]*resources.TagValidationRule `json:"tag_validation_rules,omitempty"`
}
