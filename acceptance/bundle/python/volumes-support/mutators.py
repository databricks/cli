from dataclasses import replace

from databricks.bundles.core import volume_mutator
from databricks.bundles.volumes import Volume


@volume_mutator
def update_volume(volume: Volume) -> Volume:
    assert isinstance(volume.name, str)

    return replace(volume, name=f"{volume.name} (updated)")
