package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/utils"
	sdkexperiments "github.com/databricks/sdk-go/experiments/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type MlflowExperiment struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	experiments.CreateExperimentRequest

	Permissions []MlflowExperimentPermission `json:"permissions,omitempty"`
}

func (s *MlflowExperiment) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.Experiments.GetExperiment(ctx, sdkexperiments.GetExperimentRequest{
		ExperimentId: new(id),
	})
	if err != nil {
		log.Debugf(ctx, "experiment %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (s *MlflowExperiment) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "experiment",
		PluralName:    "experiments",
		SingularTitle: "Experiment",
		PluralTitle:   "Experiments",
	}
}

func (s *MlflowExperiment) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "experiments", s.ID)
}

func (s *MlflowExperiment) GetName() string {
	return utils.Deref(s.Name)
}
