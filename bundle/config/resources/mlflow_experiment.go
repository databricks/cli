package resources

import "github.com/databricks/databricks-sdk-go/service/ml"

type MlflowExperiment struct {
	Permissions []Permission `json:"permissions,omitempty"`

	Paths

	*ml.Experiment
}
