package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowExperimentPermissionLevel string

// MlflowExperimentPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any experiment.
type MlflowExperimentPermission struct {
	Level MlflowExperimentPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type MlflowExperiment struct {
	BaseResource
	ml.CreateExperiment

	Permissions []MlflowExperimentPermission `json:"permissions,omitempty"`
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
	baseURL.Path = "ml/experiments/" + s.ID
	s.URL = baseURL.String()
}

func (s *MlflowExperiment) GetName() string {
	return s.Name
}

func (s *MlflowExperiment) GetURL() string {
	return s.URL
}
