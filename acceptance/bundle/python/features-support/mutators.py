from dataclasses import replace

from databricks.bundles.core import feature_mutator
from databricks.bundles.features import Feature


@feature_mutator
def update_feature(feature: Feature) -> Feature:
    assert isinstance(feature.description, str)

    return replace(feature, description=f"{feature.description} (updated)")
