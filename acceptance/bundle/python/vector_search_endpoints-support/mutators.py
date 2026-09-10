from dataclasses import replace

from databricks.bundles.core import vector_search_endpoint_mutator
from databricks.bundles.vector_search_endpoints import VectorSearchEndpoint


@vector_search_endpoint_mutator
def update_vector_search_endpoint(
    endpoint: VectorSearchEndpoint,
) -> VectorSearchEndpoint:
    assert isinstance(endpoint.name, str)

    return replace(endpoint, name=f"{endpoint.name} (updated)")
