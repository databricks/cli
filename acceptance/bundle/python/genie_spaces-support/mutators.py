from dataclasses import replace

from databricks.bundles.core import genie_space_mutator
from databricks.bundles.genie_spaces import GenieSpace


@genie_space_mutator
def update_genie_space(genie_space: GenieSpace) -> GenieSpace:
    assert isinstance(genie_space.description, str)

    return replace(genie_space, description=f"{genie_space.description} (updated)")
