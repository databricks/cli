package resources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowExperiment struct {
	// dynamic value representation of the resource.
	v dyn.Value

	ID             string         `json:"id,omitempty" bundle:"readonly"`
	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`

	*ml.Experiment
}

func (s *MlflowExperiment) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s MlflowExperiment) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *MlflowExperiment) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Experiments.GetExperiment(ctx, ml.GetExperimentRequest{
		ExperimentId: id,
	})
	if err != nil {
		log.Debugf(ctx, "experiment %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (s *MlflowExperiment) TerraformResourceName() string {
	return "databricks_mlflow_experiment"
}

func (s *MlflowExperiment) Validate() error {
	if s == nil || !s.v.IsValid() {
		return fmt.Errorf("experiment is not defined")
	}

	return nil
}
