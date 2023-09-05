package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

type MlflowExperiment struct {
	Permissions []Permission `json:"permissions,omitempty"`

	paths.Paths

	*ml.Experiment
}
