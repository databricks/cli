package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	marshal "github.com/databricks/databricks-sdk-go/json"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowExperiment struct {
	Permissions []Permission `json:"permissions,omitempty"`

	paths.Paths

	*ml.Experiment
}

func (s *MlflowExperiment) UnmarshalJSON(b []byte) error {
	type C MlflowExperiment
	return marshal.Unmarshal(b, (*C)(s))
}

func (s MlflowExperiment) MarshalJSON() ([]byte, error) {
	type C MlflowExperiment
	return marshal.Marshal((C)(s))
}
