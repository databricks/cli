from dataclasses import replace

from databricks.bundles.model_services import ModelService
from databricks.bundles.core import model_service_mutator


@model_service_mutator
def update_model_service(model_service: ModelService) -> ModelService:
    assert isinstance(model_service.comment, str)

    return replace(model_service, comment=f"{model_service.comment} (updated)")
