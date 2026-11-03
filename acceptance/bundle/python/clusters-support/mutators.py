from dataclasses import replace

from databricks.bundles.clusters import Cluster
from databricks.bundles.core import cluster_mutator


@cluster_mutator
def update_cluster(cluster: Cluster) -> Cluster:
    assert isinstance(cluster.cluster_name, str)

    return replace(cluster, cluster_name=f"{cluster.cluster_name} (updated)")
