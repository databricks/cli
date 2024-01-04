package config

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Mode string

// Target defines overrides for a single target.
// This structure is recursively merged into the root configuration.
type Target struct {
	// Default marks that this target must be used if one isn't specified
	// by the user (through target variable or command line argument).
	Default bool `json:"default,omitempty"`

	// Determines the mode of the target.
	// For example, 'mode: development' can be used for deployments for
	// development purposes.
	Mode Mode `json:"mode,omitempty"`

	// Overrides the compute used for jobs and other supported assets.
	ComputeID string `json:"compute_id,omitempty"`

	Bundle *Bundle `json:"bundle,omitempty"`

	Workspace *Workspace `json:"workspace,omitempty"`

	Artifacts Artifacts `json:"artifacts,omitempty"`

	Resources *Resources `json:"resources,omitempty"`

	// Override default values or lookup name for defined variables
	// Does not permit defining new variables or redefining existing ones
	// in the scope of an target
	Variables map[string]any `json:"variables,omitempty"`

	Git Git `json:"git,omitempty"`

	RunAs *jobs.JobRunAs `json:"run_as,omitempty"`

	Sync *Sync `json:"sync,omitempty"`

	Permissions []resources.Permission `json:"permissions,omitempty"`
}

const (
	// Development mode: deployments done purely for running things in development.
	// Any deployed resources will be marked as "dev" and might be hidden or cleaned up.
	Development Mode = "development"

	// Production mode: deployments done for production purposes.
	// Any deployed resources will not be changed but this mode will enable
	// various strictness checks to make sure that a deployment is correctly setup
	// for production purposes.
	Production Mode = "production"
)
