package resources

import (
	"context"
	"net/url"
	"path"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowExperiment struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
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

func (*MlflowExperiment) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*MlflowExperiment) Approval() Approval {
	return Approval{}
}

func (s *MlflowExperiment) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "experiment",
		PluralName:    "experiments",
		SingularTitle: "Experiment",
		PluralTitle:   "Experiments",
	}
}

func (*MlflowExperiment) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_READ",
		},
		RequestObjectType:  "experiments",
		ResponseObjectType: "mlflowExperiment",
		ObjectIDField:      "id",
	}
}

func (*MlflowExperiment) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "ml/experiments/%s"}, true
}

func (s *MlflowExperiment) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceURL(s, baseURL, s.ID)
}

func (s *MlflowExperiment) GetName() string {
	return s.Name
}

func (s *MlflowExperiment) ApplyPresets(p *PresetSettings) {
	filepath := s.Name
	dir := path.Dir(filepath)
	base := path.Base(filepath)
	if dir == "." {
		s.Name = p.NamePrefix + base
	} else {
		s.Name = dir + "/" + p.NamePrefix + base
	}
	for _, t := range p.Tags {
		exists := false
		for _, experimentTag := range s.Tags {
			if experimentTag.Key == t.Key {
				exists = true
				break
			}
		}
		if !exists {
			s.Tags = append(s.Tags, ml.ExperimentTag{Key: t.Key, Value: t.Value})
		}
	}
}
