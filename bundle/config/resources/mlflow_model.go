package resources

import (
	"context"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowModel struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`

	*ml.Model
}

func (s *MlflowModel) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s MlflowModel) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *MlflowModel) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.ModelRegistry.GetModel(ctx, ml.GetModelRequest{
		Name: id,
	})
	if err != nil {
		log.Debugf(ctx, "model %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (s *MlflowModel) TerraformResourceName() string {
	return "databricks_mlflow_model"
}
