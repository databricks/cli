package terranova

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/terraform-provider-databricks/catalog"
)

type IResource interface {
	// Called during init to fill in defaults and validate configuration
	Initialize()

	// Pre-process dyn.Value from configuration before using it for any subsequent operations
	PreprocessConfig(dyn.Value) (dyn.Value, error)

	// Extract ID from the configuration, if present. If not present, return "".
	ExtractIDFromConfig(dyn.Value) (string, error)

	// Create the resource with given configuration.
	// This function is called if there is state recorded for a given resource.
	// resourceID may be provided - it is the value that was returned by ExtractIDFromConfig and used to create an entry in the DB.
	DoCreate(ctx context.Context, resourceID string, config dyn.Value, client *databricks.WorkspaceClient) (string, error)

	// Update the resource with given configuration.
	DoUpdate(ctx context.Context, resourceID string, configOld, config dyn.Value, client *databricks.WorkspaceClient) error
}

var specs = map[string]IResource{
	"jobs": &ResourceSpec{
		Create: CallSpec{
			Method:          "POST",
			Path:            "/api/2.2/jobs/create",
			ResponseIDField: "job_id",
		},
		Update: CallSpec{
			Method:           "POST",
			Path:             "/api/2.2/jobs/reset",
			RequestDataField: "new_settings",
			RequestIDField:   "job_id",
		},
		Delete: CallSpec{
			Method:         "POST",
			Path:           "/api/2.2/jobs/delete",
			RequestIDField: "job_id",
		},
		Read: CallSpec{
			Method:            "GET",
			Path:              "/api/2.2/jobs/get",
			QueryIDField:      "job_id",
			ResponseDataField: "settings",
			// XXX Max 100 items, returns next_page_token for more.
		},
		Processors: []Processor{
			{
				Drop: []string{"permissions"},
			},
		},
	},
	"pipelines": &ResourceSpec{
		DefaultPath: "/api/2.0/pipelines/{}",
		Create: CallSpec{
			Method:          "POST",
			Path:            "/api/2.0/pipelines",
			ResponseIDField: "pipeline_id",
		},
		Update:                CallSpec{Method: "PUT"},
		Delete:                CallSpec{Method: "DELETE"},
		Read:                  CallSpec{Method: "GET"},
		ReadinessField:        "state",
		ReadinessFieldSuccess: []string{"RUNNING"},
		ReadinessFieldFailure: []string{"FAILED"},
		ReadinessEval: func(spec *ResourceSpec, value dyn.Value) (bool, error) {
			isReady, err := DefaultReadinessEval(spec, value)
			if isReady {
				return isReady, err
			}
			continuous, ok := dyn.GetByString(value, "spec.continuous").AsBool()
			if !ok {
				return true, errors.New("Cannot parse spec.continuous field: missing or wrong type")
			}
			if !continuous {
				// following terraform:
				// https://github.com/databricks/terraform-provider-databricks/blob/a76703c037/pipelines/resource_pipeline.go#L124
				return true, nil
			}
			return false, nil
		},
	},
	"quality_monitors": &ResourceSpec{
		DefaultPath:           "/api/2.1/unity-catalog/tables/{}/monitor",
		ConfigIDField:         "table_name",
		Create:                CallSpec{Method: "POST"},
		Update:                CallSpec{Method: "PUT"},
		Delete:                CallSpec{Method: "DELETE"},
		Read:                  CallSpec{Method: "GET"},
		ReadinessField:        "status",
		ReadinessFieldSuccess: []string{"MONITOR_STATUS_ACTIVE"},
		ReadinessFieldFailure: []string{
			"MONITOR_STATUS_ERROR",
			"MONITOR_STATUS_FAILED",
		},
	},
	"apps": &ResourceSpec{
		DefaultPath:   "/api/2.0/apps/{}",
		ConfigIDField: "name",
		Processors: []Processor{
			{
				Drop: []string{"permissions", "source_code_path"},
			},
		},
		Create: CallSpec{
			Method: "POST",
			Path:   "/api/2.0/apps",
		},
		Update:                CallSpec{Method: "PATCH"},
		Delete:                CallSpec{Method: "DELETE"},
		Read:                  CallSpec{Method: "GET"},
		ReadinessField:        "app_status.state",
		ReadinessFieldSuccess: []string{"RUNNING"},
		ReadinessFieldFailure: []string{
			"CRASHED",
			"UNAVAILABLE",
		},
	},

	// terraform SDKv2 wrappers:

	"schemas": &TerraformWrapper{
		ConverterName:  "schemas",
		CommonResource: catalog.ResourceSchema(),
		ExtractIDFunc: func(t *TerraformWrapper, config dyn.Value) (string, error) {
			catalog_name, err := GetRequiredNonemptyString(config, "catalog_name")
			if err != nil {
				return "", err
			}

			name, err := GetRequiredNonemptyString(config, "name")
			if err != nil {
				return "", err
			}

			return catalog_name + "." + name, nil
		},
	},
}

func init() {
	for _, r := range specs {
		r.Initialize()
	}
}
