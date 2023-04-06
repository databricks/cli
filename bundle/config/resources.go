package config

import (
	"github.com/databricks/bricks/bundle/config/resources"
)

// Resources defines Databricks resources associated with the bundle.
type Resources struct {
	Jobs      map[string]*resources.Job      `json:"jobs,omitempty"`
	Pipelines map[string]*resources.Pipeline `json:"pipelines,omitempty"`

	Models      map[string]*resources.MlflowModel      `json:"models,omitempty"`
	Experiments map[string]*resources.MlflowExperiment `json:"experiments,omitempty"`
}

// SetConfigFilePath sets the specified path for all resources contained in this instance.
// This property is used to correctly resolve paths relative to the path
// of the configuration file they were defined in.
func (r *Resources) SetConfigFilePath(path string) {
	for _, e := range r.Jobs {
		e.ConfigFilePath = path
	}
	for _, e := range r.Pipelines {
		e.ConfigFilePath = path
	}
	for _, e := range r.Models {
		e.ConfigFilePath = path
	}
	for _, e := range r.Experiments {
		e.ConfigFilePath = path
	}
}
