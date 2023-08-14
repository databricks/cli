package config

import (
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
)

// Resources defines Databricks resources associated with the bundle.
type Resources struct {
	Jobs      map[string]*resources.Job      `json:"jobs,omitempty"`
	Pipelines map[string]*resources.Pipeline `json:"pipelines,omitempty"`

	Models      map[string]*resources.MlflowModel      `json:"models,omitempty"`
	Experiments map[string]*resources.MlflowExperiment `json:"experiments,omitempty"`
}

type UniqueResourceIdTracker struct {
	Type       map[string]string
	ConfigPath map[string]string
}

// verifies merging is safe by checking no duplicate identifiers exist
func (r *Resources) VerifySafeMerge(other *Resources) error {
	rootTracker, err := r.VerifyUniqueResourceIdentifiers()
	if err != nil {
		return err
	}
	otherTracker, err := other.VerifyUniqueResourceIdentifiers()
	if err != nil {
		return err
	}
	for k := range otherTracker.Type {
		if _, ok := rootTracker.Type[k]; ok {
			return fmt.Errorf("multiple resources named %s (%s at %s, %s at %s)",
				k,
				rootTracker.Type[k],
				rootTracker.ConfigPath[k],
				otherTracker.Type[k],
				otherTracker.ConfigPath[k],
			)
		}
	}
	return nil
}

// This function verifies there are no duplicate names used for the resource definations
func (r *Resources) VerifyUniqueResourceIdentifiers() (*UniqueResourceIdTracker, error) {
	tracker := &UniqueResourceIdTracker{
		Type:       make(map[string]string),
		ConfigPath: make(map[string]string),
	}
	for k := range r.Jobs {
		tracker.Type[k] = "job"
		tracker.ConfigPath[k] = r.Jobs[k].ConfigFilePath
	}
	for k := range r.Pipelines {
		if _, ok := tracker.Type[k]; ok {
			return tracker, fmt.Errorf("multiple resources named %s (%s at %s, %s at %s)",
				k,
				tracker.Type[k],
				tracker.ConfigPath[k],
				"pipeline",
				r.Pipelines[k].ConfigFilePath,
			)
		}
		tracker.Type[k] = "pipeline"
		tracker.ConfigPath[k] = r.Pipelines[k].ConfigFilePath
	}
	for k := range r.Models {
		if _, ok := tracker.Type[k]; ok {
			return tracker, fmt.Errorf("multiple resources named %s (%s at %s, %s at %s)",
				k,
				tracker.Type[k],
				tracker.ConfigPath[k],
				"mlflow_model",
				r.Models[k].ConfigFilePath,
			)
		}
		tracker.Type[k] = "mlflow_model"
		tracker.ConfigPath[k] = r.Models[k].ConfigFilePath
	}
	for k := range r.Experiments {
		if _, ok := tracker.Type[k]; ok {
			return tracker, fmt.Errorf("multiple resources named %s (%s at %s, %s at %s)",
				k,
				tracker.Type[k],
				tracker.ConfigPath[k],
				"mlflow_experiment",
				r.Experiments[k].ConfigFilePath,
			)
		}
		tracker.Type[k] = "mlflow_experiment"
		tracker.ConfigPath[k] = r.Experiments[k].ConfigFilePath
	}
	return tracker, nil
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

// MergeJobClusters iterates over all jobs and merges their job clusters.
// This is called after applying the environment overrides.
func (r *Resources) MergeJobClusters() error {
	for _, job := range r.Jobs {
		if err := job.MergeJobClusters(); err != nil {
			return err
		}
	}
	return nil
}
