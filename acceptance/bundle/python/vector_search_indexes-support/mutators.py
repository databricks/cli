from dataclasses import replace

from databricks.bundles.core import vector_search_index_mutator
from databricks.bundles.vector_search_indexes import VectorSearchIndex


@vector_search_index_mutator
def update_vector_search_index(index: VectorSearchIndex) -> VectorSearchIndex:
    assert isinstance(index.name, str)

    return replace(index, name=f"{index.name} (updated)")
