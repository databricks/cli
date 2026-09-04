from dataclasses import replace

from databricks.bundles.models import MlflowModel
from databricks.bundles.core import mlflow_model_mutator


@mlflow_model_mutator
def update_mlflow_model(model: MlflowModel) -> MlflowModel:
    assert isinstance(model.name, str)

    return replace(model, name=f"{model.name} (updated)")
