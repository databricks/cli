from dataclasses import replace

from databricks.bundles.core import instance_pool_mutator
from databricks.bundles.instance_pools import InstancePool


@instance_pool_mutator
def update_instance_pool(pool: InstancePool) -> InstancePool:
    assert isinstance(pool.instance_pool_name, str)

    return replace(pool, instance_pool_name=f"{pool.instance_pool_name} (updated)")
