from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.aws_attributes import (
    AwsAttributes,
    AwsAttributesParam,
)
from databricks.bundles.compute._models.azure_attributes import (
    AzureAttributes,
    AzureAttributesParam,
)
from databricks.bundles.compute._models.cluster_log_conf import (
    ClusterLogConf,
    ClusterLogConfParam,
)
from databricks.bundles.compute._models.gcp_attributes import (
    GcpAttributes,
    GcpAttributesParam,
)
from databricks.bundles.compute._models.init_script_info import (
    InitScriptInfo,
    InitScriptInfoParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.pipelines._models.pipeline_cluster_autoscale import (
    PipelineClusterAutoscale,
    PipelineClusterAutoscaleParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelineCluster:
    """"""

    apply_policy_default_values: VariableOrOptional[bool] = None
    """
    Note: This field won't be persisted. Only API users will check this field.
    """

    autoscale: VariableOrOptional[PipelineClusterAutoscale] = None
    """
    Parameters needed in order to automatically scale clusters up and down based on load.
    Note: autoscaling works best with DB runtime versions 3.0 or later.
    """

    aws_attributes: VariableOrOptional[AwsAttributes] = None
    """
    Attributes related to clusters running on Amazon Web Services.
    If not specified at cluster creation, a set of default values will be used.
    """

    azure_attributes: VariableOrOptional[AzureAttributes] = None
    """
    Attributes related to clusters running on Microsoft Azure.
    If not specified at cluster creation, a set of default values will be used.
    """

    cluster_log_conf: VariableOrOptional[ClusterLogConf] = None
    """
    The configuration for delivering spark logs to a long-term storage destination.
    Only dbfs destinations are supported. Only one destination can be specified
    for one cluster. If the conf is given, the logs will be delivered to the destination every
    `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
    the destination of executor logs is `$destination/$clusterId/executor`.
    """

    custom_tags: VariableOrDict[str] = field(default_factory=dict)
    """
    Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
    instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
    
    - Currently, Databricks allows at most 45 custom tags
    
    - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
    """

    driver_instance_pool_id: VariableOrOptional[str] = None
    """
    The optional ID of the instance pool for the driver of the cluster belongs.
    The pool cluster uses the instance pool with id (instance_pool_id) if the driver pool is not
    assigned.
    """

    driver_node_type_id: VariableOrOptional[str] = None
    """
    The node type of the Spark driver.
    Note that this field is optional; if unset, the driver node type will be set as the same value
    as `node_type_id` defined above.
    """

    enable_local_disk_encryption: VariableOrOptional[bool] = None
    """
    Whether to enable local disk encryption for the cluster.
    """

    gcp_attributes: VariableOrOptional[GcpAttributes] = None
    """
    Attributes related to clusters running on Google Cloud Platform.
    If not specified at cluster creation, a set of default values will be used.
    """

    init_scripts: VariableOrList[InitScriptInfo] = field(default_factory=list)
    """
    The configuration for storing init scripts. Any number of destinations can be specified. The scripts are executed sequentially in the order provided. If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
    """

    instance_pool_id: VariableOrOptional[str] = None
    """
    The optional ID of the instance pool to which the cluster belongs.
    """

    label: VariableOrOptional[str] = None
    """
    A label for the cluster specification, either `default` to configure the default cluster, or `maintenance` to configure the maintenance cluster. This field is optional. The default value is `default`.
    """

    node_type_id: VariableOrOptional[str] = None
    """
    This field encodes, through a single value, the resources available to each of
    the Spark nodes in this cluster. For example, the Spark nodes can be provisioned
    and optimized for memory or compute intensive workloads. A list of available node
    types can be retrieved by using the :method:clusters/listNodeTypes API call.
    """

    num_workers: VariableOrOptional[int] = None
    """
    Number of worker nodes that this cluster should have. A cluster has one Spark Driver
    and `num_workers` Executors for a total of `num_workers` + 1 Spark nodes.
    
    Note: When reading the properties of a cluster, this field reflects the desired number
    of workers rather than the actual current number of workers. For instance, if a cluster
    is resized from 5 to 10 workers, this field will immediately be updated to reflect
    the target size of 10 workers, whereas the workers listed in `spark_info` will gradually
    increase from 5 to 10 as the new nodes are provisioned.
    """

    policy_id: VariableOrOptional[str] = None
    """
    The ID of the cluster policy used to create the cluster if applicable.
    """

    spark_conf: VariableOrDict[str] = field(default_factory=dict)
    """
    An object containing a set of optional, user-specified Spark configuration key-value pairs.
    See :method:clusters/create for more details.
    """

    spark_env_vars: VariableOrDict[str] = field(default_factory=dict)
    """
    An object containing a set of optional, user-specified environment variable key-value pairs.
    Please note that key-value pair of the form (X,Y) will be exported as is (i.e.,
    `export X='Y'`) while launching the driver and workers.
    
    In order to specify an additional set of `SPARK_DAEMON_JAVA_OPTS`, we recommend appending
    them to `$SPARK_DAEMON_JAVA_OPTS` as shown in the example below. This ensures that all
    default databricks managed environmental variables are included as well.
    
    Example Spark environment variables:
    `{"SPARK_WORKER_MEMORY": "28000m", "SPARK_LOCAL_DIRS": "/local_disk0"}` or
    `{"SPARK_DAEMON_JAVA_OPTS": "$SPARK_DAEMON_JAVA_OPTS -Dspark.shuffle.service.enabled=true"}`
    """

    ssh_public_keys: VariableOrList[str] = field(default_factory=list)
    """
    SSH public key contents that will be added to each Spark node in this cluster. The
    corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
    Up to 10 keys can be specified.
    """

    @classmethod
    def from_dict(cls, value: "PipelineClusterDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineClusterDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineClusterDict(TypedDict, total=False):
    """"""

    apply_policy_default_values: VariableOrOptional[bool]
    """
    Note: This field won't be persisted. Only API users will check this field.
    """

    autoscale: VariableOrOptional[PipelineClusterAutoscaleParam]
    """
    Parameters needed in order to automatically scale clusters up and down based on load.
    Note: autoscaling works best with DB runtime versions 3.0 or later.
    """

    aws_attributes: VariableOrOptional[AwsAttributesParam]
    """
    Attributes related to clusters running on Amazon Web Services.
    If not specified at cluster creation, a set of default values will be used.
    """

    azure_attributes: VariableOrOptional[AzureAttributesParam]
    """
    Attributes related to clusters running on Microsoft Azure.
    If not specified at cluster creation, a set of default values will be used.
    """

    cluster_log_conf: VariableOrOptional[ClusterLogConfParam]
    """
    The configuration for delivering spark logs to a long-term storage destination.
    Only dbfs destinations are supported. Only one destination can be specified
    for one cluster. If the conf is given, the logs will be delivered to the destination every
    `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
    the destination of executor logs is `$destination/$clusterId/executor`.
    """

    custom_tags: VariableOrDict[str]
    """
    Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
    instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
    
    - Currently, Databricks allows at most 45 custom tags
    
    - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
    """

    driver_instance_pool_id: VariableOrOptional[str]
    """
    The optional ID of the instance pool for the driver of the cluster belongs.
    The pool cluster uses the instance pool with id (instance_pool_id) if the driver pool is not
    assigned.
    """

    driver_node_type_id: VariableOrOptional[str]
    """
    The node type of the Spark driver.
    Note that this field is optional; if unset, the driver node type will be set as the same value
    as `node_type_id` defined above.
    """

    enable_local_disk_encryption: VariableOrOptional[bool]
    """
    Whether to enable local disk encryption for the cluster.
    """

    gcp_attributes: VariableOrOptional[GcpAttributesParam]
    """
    Attributes related to clusters running on Google Cloud Platform.
    If not specified at cluster creation, a set of default values will be used.
    """

    init_scripts: VariableOrList[InitScriptInfoParam]
    """
    The configuration for storing init scripts. Any number of destinations can be specified. The scripts are executed sequentially in the order provided. If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
    """

    instance_pool_id: VariableOrOptional[str]
    """
    The optional ID of the instance pool to which the cluster belongs.
    """

    label: VariableOrOptional[str]
    """
    A label for the cluster specification, either `default` to configure the default cluster, or `maintenance` to configure the maintenance cluster. This field is optional. The default value is `default`.
    """

    node_type_id: VariableOrOptional[str]
    """
    This field encodes, through a single value, the resources available to each of
    the Spark nodes in this cluster. For example, the Spark nodes can be provisioned
    and optimized for memory or compute intensive workloads. A list of available node
    types can be retrieved by using the :method:clusters/listNodeTypes API call.
    """

    num_workers: VariableOrOptional[int]
    """
    Number of worker nodes that this cluster should have. A cluster has one Spark Driver
    and `num_workers` Executors for a total of `num_workers` + 1 Spark nodes.
    
    Note: When reading the properties of a cluster, this field reflects the desired number
    of workers rather than the actual current number of workers. For instance, if a cluster
    is resized from 5 to 10 workers, this field will immediately be updated to reflect
    the target size of 10 workers, whereas the workers listed in `spark_info` will gradually
    increase from 5 to 10 as the new nodes are provisioned.
    """

    policy_id: VariableOrOptional[str]
    """
    The ID of the cluster policy used to create the cluster if applicable.
    """

    spark_conf: VariableOrDict[str]
    """
    An object containing a set of optional, user-specified Spark configuration key-value pairs.
    See :method:clusters/create for more details.
    """

    spark_env_vars: VariableOrDict[str]
    """
    An object containing a set of optional, user-specified environment variable key-value pairs.
    Please note that key-value pair of the form (X,Y) will be exported as is (i.e.,
    `export X='Y'`) while launching the driver and workers.
    
    In order to specify an additional set of `SPARK_DAEMON_JAVA_OPTS`, we recommend appending
    them to `$SPARK_DAEMON_JAVA_OPTS` as shown in the example below. This ensures that all
    default databricks managed environmental variables are included as well.
    
    Example Spark environment variables:
    `{"SPARK_WORKER_MEMORY": "28000m", "SPARK_LOCAL_DIRS": "/local_disk0"}` or
    `{"SPARK_DAEMON_JAVA_OPTS": "$SPARK_DAEMON_JAVA_OPTS -Dspark.shuffle.service.enabled=true"}`
    """

    ssh_public_keys: VariableOrList[str]
    """
    SSH public key contents that will be added to each Spark node in this cluster. The
    corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
    Up to 10 keys can be specified.
    """


PipelineClusterParam = PipelineClusterDict | PipelineCluster
