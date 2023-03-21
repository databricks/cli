package resources

import "github.com/databricks/databricks-sdk-go/service/mlflow"

type MlflowModel struct {
	Permissions []Permission `json:"permissions,omitempty"`

	*mlflow.RegisteredModel
}
