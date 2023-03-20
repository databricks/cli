package resources

import "github.com/databricks/databricks-sdk-go/service/mlflow"

type MlflowModel struct {
	*mlflow.RegisteredModel
}
