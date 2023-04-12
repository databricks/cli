package resources

import "github.com/databricks/databricks-sdk-go/service/mlflow"

type MlflowExperiment struct {
	Permissions []Permission `json:"permissions,omitempty"`

	Paths

	*mlflow.Experiment
}
