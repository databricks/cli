from dataclasses import replace

from databricks.bundles.core import materialization_set_mutator
from databricks.bundles.materialization_sets import MaterializationSet


@materialization_set_mutator
def update_materialization_set(
    materialization_set: MaterializationSet,
) -> MaterializationSet:
    # A set has no string field of its own, so the member's feature name is what a mutator can
    # observably change.
    members = list(materialization_set.materialized_features)
    first = members[0]
    assert isinstance(first.feature_name, str)

    return replace(
        materialization_set,
        materialized_features=[
            replace(first, feature_name=f"{first.feature_name}_updated"),
            *members[1:],
        ],
    )
