package mlflow

import (
	"github.com/databricks/bricks/cmd/mlflow/experiments"
	m_lflow_artifacts "github.com/databricks/bricks/cmd/mlflow/m-lflow-artifacts"
	m_lflow_databricks "github.com/databricks/bricks/cmd/mlflow/m-lflow-databricks"
	m_lflow_metrics "github.com/databricks/bricks/cmd/mlflow/m-lflow-metrics"
	m_lflow_runs "github.com/databricks/bricks/cmd/mlflow/m-lflow-runs"
	model_version_comments "github.com/databricks/bricks/cmd/mlflow/model-version-comments"
	model_versions "github.com/databricks/bricks/cmd/mlflow/model-versions"
	registered_models "github.com/databricks/bricks/cmd/mlflow/registered-models"
	registry_webhooks "github.com/databricks/bricks/cmd/mlflow/registry-webhooks"
	transition_requests "github.com/databricks/bricks/cmd/mlflow/transition-requests"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "mlflow",
}

func init() {

	Cmd.AddCommand(experiments.Cmd)
	Cmd.AddCommand(m_lflow_artifacts.Cmd)
	Cmd.AddCommand(m_lflow_databricks.Cmd)
	Cmd.AddCommand(m_lflow_metrics.Cmd)
	Cmd.AddCommand(m_lflow_runs.Cmd)
	Cmd.AddCommand(model_version_comments.Cmd)
	Cmd.AddCommand(model_versions.Cmd)
	Cmd.AddCommand(registered_models.Cmd)
	Cmd.AddCommand(registry_webhooks.Cmd)
	Cmd.AddCommand(transition_requests.Cmd)
}
