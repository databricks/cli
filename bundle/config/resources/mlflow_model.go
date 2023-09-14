package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowModel struct {
	Permissions []Permission `json:"permissions,omitempty"`

	paths.Paths

	*ml.Model
}

func (s *MlflowModel) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s MlflowModel) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}
