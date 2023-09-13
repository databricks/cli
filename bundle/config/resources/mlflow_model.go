package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	marshal "github.com/databricks/databricks-sdk-go/json"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowModel struct {
	Permissions []Permission `json:"permissions,omitempty"`

	paths.Paths

	*ml.Model
}

func (s *MlflowModel) UnmarshalJSON(b []byte) error {
	type C MlflowModel
	return marshal.Unmarshal(b, (*C)(s))
}

func (s MlflowModel) MarshalJSON() ([]byte, error) {
	type C MlflowModel
	return marshal.Marshal((C)(s))
}
