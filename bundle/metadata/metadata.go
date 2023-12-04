package metadata

import (
	"github.com/databricks/cli/bundle/config"
)

const Version = 1

type Bundle struct {
	Git config.Git `json:"git,omitempty"`
}

type Workspace struct {
	FilePath string `json:"file_path"`
}

type Job struct {
	ID string `json:"id,omitempty"`

	// Relative path from the bundle root to the configuration file that holds
	// the definition of this resource.
	RelativePath string `json:"relative_path"`
}

type Resources struct {
	Jobs map[string]*Job `json:"jobs,omitempty"`
}

type Config struct {
	Bundle    Bundle    `json:"bundle,omitempty"`
	Workspace Workspace `json:"workspace,omitempty"`
	Resources Resources `json:"resources,omitempty"`
}

// Metadata about the bundle deployment. This is the interface Databricks services
// rely on to integrate with bundles when they need additional information about
// a bundle deployment.
//
// After deploy, a file containing the metadata (metadata.json) can be found
// in the WSFS location containing the bundle state.
type Metadata struct {
	Version int `json:"version"`

	Config Config `json:"config"`
}
