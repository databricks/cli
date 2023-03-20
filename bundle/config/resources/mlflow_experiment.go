package resources

import "github.com/databricks/databricks-sdk-go/service/mlflow"

type MlflowExperiment struct {
	ID string `json:"id,omitempty"`

	*mlflow.Experiment
}
