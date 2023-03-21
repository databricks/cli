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
