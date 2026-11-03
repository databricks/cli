from dataclasses import replace

from databricks.bundles.model_serving_endpoints import ModelServingEndpoint
from databricks.bundles.core import model_serving_endpoint_mutator


@model_serving_endpoint_mutator
def update_model_serving_endpoint(endpoint: ModelServingEndpoint) -> ModelServingEndpoint:
    assert isinstance(endpoint.name, str)

    return replace(endpoint, name=f"{endpoint.name} (updated)")
