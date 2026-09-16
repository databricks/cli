from dataclasses import replace

from databricks.bundles.core import registered_model_mutator
from databricks.bundles.registered_models import RegisteredModel


@registered_model_mutator
def update_registered_model(registered_model: RegisteredModel) -> RegisteredModel:
    assert isinstance(registered_model.comment, str)

    return replace(registered_model, comment=f"{registered_model.comment} (updated)")
