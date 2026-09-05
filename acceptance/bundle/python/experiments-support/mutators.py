from dataclasses import replace

from databricks.bundles.experiments import MlflowExperiment
from databricks.bundles.core import mlflow_experiment_mutator


@mlflow_experiment_mutator
def update_mlflow_experiment(experiment: MlflowExperiment) -> MlflowExperiment:
    assert isinstance(experiment.name, str)

    return replace(experiment, name=f"{experiment.name} (updated)")
