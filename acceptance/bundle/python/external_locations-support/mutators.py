from dataclasses import replace

from databricks.bundles.core import external_location_mutator
from databricks.bundles.external_locations import ExternalLocation


@external_location_mutator
def update_external_location(location: ExternalLocation) -> ExternalLocation:
    assert isinstance(location.name, str)

    return replace(location, name=f"{location.name} (updated)")
