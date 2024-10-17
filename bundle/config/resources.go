package config

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
)

// Resources defines Databricks resources associated with the bundle.
type Resources struct {
	Jobs      map[string]*resources.Job      `json:"jobs,omitempty"`
	Pipelines map[string]*resources.Pipeline `json:"pipelines,omitempty"`

	Models                map[string]*resources.MlflowModel          `json:"models,omitempty"`
	Experiments           map[string]*resources.MlflowExperiment     `json:"experiments,omitempty"`
	ModelServingEndpoints map[string]*resources.ModelServingEndpoint `json:"model_serving_endpoints,omitempty"`
	RegisteredModels      map[string]*resources.RegisteredModel      `json:"registered_models,omitempty"`
	QualityMonitors       map[string]*resources.QualityMonitor       `json:"quality_monitors,omitempty"`
	Schemas               map[string]*resources.Schema               `json:"schemas,omitempty"`
	Clusters              map[string]*resources.Cluster              `json:"clusters,omitempty"`
}

type ConfigResource interface {
	// Function to assert if the resource exists in the workspace configured in
	// the input workspace client.
	Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error)

	// Terraform equivalent name of the resource. For example "databricks_job"
	// for jobs and "databricks_pipeline" for pipelines.
	TerraformResourceName() string

	// GetName returns the in-product name of the resource.
	GetName() string

	// GetURL returns the URL of the resource.
	GetURL() string

	// InitializeURL initializes the URL field of the resource.
	InitializeURL(urlPrefix string, urlSuffix string)
}

func (r *Resources) AllResources() map[string]map[string]ConfigResource {
	result := make(map[string]map[string]ConfigResource)

	jobResources := make(map[string]ConfigResource)
	for key, job := range r.Jobs {
		jobResources[key] = job
	}
	result["jobs"] = jobResources

	pipelineResources := make(map[string]ConfigResource)
	for key, pipeline := range r.Pipelines {
		pipelineResources[key] = pipeline
	}
	result["pipelines"] = pipelineResources

	modelResources := make(map[string]ConfigResource)
	for key, model := range r.Models {
		modelResources[key] = model
	}
	result["models"] = modelResources

	experimentResources := make(map[string]ConfigResource)
	for key, experiment := range r.Experiments {
		experimentResources[key] = experiment
	}
	result["experiments"] = experimentResources

	modelServingEndpointResources := make(map[string]ConfigResource)
	for key, endpoint := range r.ModelServingEndpoints {
		modelServingEndpointResources[key] = endpoint
	}
	result["model_serving_endpoints"] = modelServingEndpointResources

	registeredModelResources := make(map[string]ConfigResource)
	for key, registeredModel := range r.RegisteredModels {
		registeredModelResources[key] = registeredModel
	}
	result["registered_models"] = registeredModelResources

	qualityMonitorResources := make(map[string]ConfigResource)
	for key, qualityMonitor := range r.QualityMonitors {
		qualityMonitorResources[key] = qualityMonitor
	}
	result["quality_monitors"] = qualityMonitorResources

	schemaResources := make(map[string]ConfigResource)
	for key, schema := range r.Schemas {
		schemaResources[key] = schema
	}
	result["schemas"] = schemaResources

	clusterResources := make(map[string]ConfigResource)
	for key, schema := range r.Clusters {
		clusterResources[key] = schema
	}
	result["clusters"] = clusterResources

	return result
}

func (r *Resources) FindResourceByConfigKey(key string) (ConfigResource, error) {
	found := make([]ConfigResource, 0)
	for k := range r.Jobs {
		if k == key {
			found = append(found, r.Jobs[k])
		}
	}
	for k := range r.Pipelines {
		if k == key {
			found = append(found, r.Pipelines[k])
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no such resource: %s", key)
	}

	if len(found) > 1 {
		keys := make([]string, 0, len(found))
		for _, r := range found {
			keys = append(keys, fmt.Sprintf("%s:%s", r.TerraformResourceName(), key))
		}
		return nil, fmt.Errorf("ambiguous: %s (can resolve to all of %s)", key, keys)
	}

	return found[0], nil
}

type ResourceDescription struct {
	// Singular and plural name when used to refer to the configuration.
	SingularName string
	PluralName   string

	// Singular and plural title when used in summaries / terminal UI.
	SingularTitle string
	PluralTitle   string
}

// The keys of the map corresponds to the resource key in the bundle configuration.
func SupportedResources() map[string]ResourceDescription {
	return map[string]ResourceDescription{
		"jobs": {
			SingularName:  "job",
			PluralName:    "jobs",
			SingularTitle: "Job",
			PluralTitle:   "Jobs",
		},
		"pipelines": {
			SingularName:  "pipeline",
			PluralName:    "pipelines",
			SingularTitle: "Pipeline",
			PluralTitle:   "Pipelines",
		},
		"models": {
			SingularName:  "model",
			PluralName:    "models",
			SingularTitle: "Model",
			PluralTitle:   "Models",
		},
		"experiments": {
			SingularName:  "experiment",
			PluralName:    "experiments",
			SingularTitle: "Experiment",
			PluralTitle:   "Experiments",
		},
		"model_serving_endpoints": {
			SingularName:  "model_serving_endpoint",
			PluralName:    "model_serving_endpoints",
			SingularTitle: "Model Serving Endpoint",
			PluralTitle:   "Model Serving Endpoints",
		},
		"registered_models": {
			SingularName:  "registered_model",
			PluralName:    "registered_models",
			SingularTitle: "Registered Model",
			PluralTitle:   "Registered Models",
		},
		"quality_monitors": {
			SingularName:  "quality_monitor",
			PluralName:    "quality_monitors",
			SingularTitle: "Quality Monitor",
			PluralTitle:   "Quality Monitors",
		},
		"schemas": {
			SingularName:  "schema",
			PluralName:    "schemas",
			SingularTitle: "Schema",
			PluralTitle:   "Schemas",
		},
		"clusters": {
			SingularName:  "cluster",
			PluralName:    "clusters",
			SingularTitle: "Cluster",
			PluralTitle:   "Clusters",
		},
	}
}
