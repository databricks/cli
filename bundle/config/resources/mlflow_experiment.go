package resources

import (
	"context"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowExperiment struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`

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

func (s *MlflowExperiment) InitializeURL(urlPrefix string, urlSuffix string) {
	if s.ID == "" {
		return
	}
	s.URL = urlPrefix + "ml/experiments/" + s.ID + urlSuffix
}

func (s *MlflowExperiment) GetName() string {
	return s.Name
}

func (s *MlflowExperiment) GetURL() string {
	return s.URL
}
