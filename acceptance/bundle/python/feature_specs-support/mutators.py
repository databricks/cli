from dataclasses import replace

from databricks.bundles.core import feature_spec_mutator
from databricks.bundles.feature_specs import FeatureSpec


@feature_spec_mutator
def update_feature_spec(feature_spec: FeatureSpec) -> FeatureSpec:
    assert isinstance(feature_spec.name, str)

    return replace(feature_spec, name=f"{feature_spec.name}_updated")
