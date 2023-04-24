package resources

import "github.com/databricks/databricks-sdk-go/service/ml"

type MlflowModel struct {
	Permissions []Permission `json:"permissions,omitempty"`

	Paths

	*ml.Model
}
