from dataclasses import replace

from databricks.bundles.cluster_policies import ClusterPolicy
from databricks.bundles.core import cluster_policy_mutator


@cluster_policy_mutator
def update_cluster_policy(cluster_policy: ClusterPolicy) -> ClusterPolicy:
    assert isinstance(cluster_policy.description, str)

    return replace(cluster_policy, description=f"{cluster_policy.description} (updated)")
