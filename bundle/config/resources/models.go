package resources

import (
	"context"
	"net/url"
	"slices"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowModel struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	ml.CreateModelRequest

	Permissions []MlflowModelPermission `json:"permissions,omitempty"`
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

func (*MlflowModel) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*MlflowModel) Approval() Approval {
	return Approval{}
}

func (s *MlflowModel) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "model",
		PluralName:    "models",
		SingularTitle: "Model",
		PluralTitle:   "Models",
	}
}

func (*MlflowModel) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_READ",
		},
		RequestObjectType:  "registered-models",
		ResponseObjectType: "registered-model",
		ObjectIDField:      "model_id", // numeric model ID, not the model name used as CRUD state ID
	}
}

func (*MlflowModel) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "ml/models/%s"}, true
}

func (s *MlflowModel) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	s.URL = workspaceURL(s, baseURL, s.ID)
}

func (s *MlflowModel) GetName() string {
	return s.Name
}

func (s *MlflowModel) ApplyPresets(p *PresetSettings) {
	s.Name = p.NamePrefix + s.Name
	for _, t := range p.Tags {
		exists := slices.ContainsFunc(s.Tags, func(modelTag ml.ModelTag) bool {
			return modelTag.Key == t.Key
		})
		if !exists {
			// Only add this tag if the resource didn't include any tag that overrides its value.
			s.Tags = append(s.Tags, ml.ModelTag{Key: t.Key, Value: t.Value})
		}
	}
}
