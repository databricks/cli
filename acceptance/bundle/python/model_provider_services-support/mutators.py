from dataclasses import replace

from databricks.bundles.model_provider_services import ModelProviderService
from databricks.bundles.core import model_provider_service_mutator


@model_provider_service_mutator
def update_model_provider_service(
    model_provider_service: ModelProviderService,
) -> ModelProviderService:
    assert isinstance(model_provider_service.comment, str)

    return replace(
        model_provider_service,
        comment=f"{model_provider_service.comment} (updated)",
    )
