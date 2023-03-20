package resources

import "github.com/databricks/databricks-sdk-go/service/mlflow"

type MlflowExperiment struct {
	*mlflow.Experiment
}
