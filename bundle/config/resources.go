package config

import "github.com/databricks/bricks/bundle/config/resources"

// Resources defines Databricks resources associated with the bundle.
type Resources struct {
	Workflows map[string]resources.Workflow `json:"workflows,omitempty"`
	Pipelines map[string]resources.Pipeline `json:"pipelines,omitempty"`
}
