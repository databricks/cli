package resources

import "github.com/databricks/databricks-sdk-go/service/mlflow"

type MlflowModel struct {
	ID string `json:"id,omitempty"`

	*mlflow.RegisteredModelDatabricks
}
