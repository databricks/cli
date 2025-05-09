from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.auto_scale import (
    AutoScale,
    AutoScaleParam,
)
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
from databricks.bundles.compute._models.data_security_mode import (
    DataSecurityMode,
    DataSecurityModeParam,
)
from databricks.bundles.compute._models.docker_image import (
    DockerImage,
    DockerImageParam,
)
from databricks.bundles.compute._models.gcp_attributes import (
    GcpAttributes,
    GcpAttributesParam,
)
from databricks.bundles.compute._models.init_script_info import (
    InitScriptInfo,
    InitScriptInfoParam,
)
from databricks.bundles.compute._models.runtime_engine import (
    RuntimeEngine,
    RuntimeEngineParam,
)
from databricks.bundles.compute._models.workload_type import (
    WorkloadType,
    WorkloadTypeParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ClusterSpec:
    """
    Contains a snapshot of the latest user specified settings that were used to create/edit the cluster.
    """

    apply_policy_default_values: VariableOrOptional[bool] = None
    """
    When set to true, fixed and default values from the policy will be used for fields that are omitted. When set to false, only fixed values from the policy will be applied.
    """

    autoscale: VariableOrOptional[AutoScale] = None
    """
    Parameters needed in order to automatically scale clusters up and down based on load.
    Note: autoscaling works best with DB runtime versions 3.0 or later.
    """

    autotermination_minutes: VariableOrOptional[int] = None
    """
    Automatically terminates the cluster after it is inactive for this time in minutes. If not set,
    this cluster will not be automatically terminated. If specified, the threshold must be between
    10 and 10000 minutes.
    Users can also set this value to 0 to explicitly disable automatic termination.
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
    Three kinds of destinations (DBFS, S3 and Unity Catalog volumes) are supported. Only one destination can be specified
    for one cluster. If the conf is given, the logs will be delivered to the destination every
    `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
    the destination of executor logs is `$destination/$clusterId/executor`.
    """

    cluster_name: VariableOrOptional[str] = None
    """
    Cluster name requested by the user. This doesn't have to be unique.
    If not specified at creation, the cluster name will be an empty string.
    """

    custom_tags: VariableOrDict[str] = field(default_factory=dict)
    """
    Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
    instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
    
    - Currently, Databricks allows at most 45 custom tags
    
    - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
    """

    data_security_mode: VariableOrOptional[DataSecurityMode] = None

    docker_image: VariableOrOptional[DockerImage] = None

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
    
    This field, along with node_type_id, should not be set if virtual_cluster_size is set.
    If both driver_node_type_id, node_type_id, and virtual_cluster_size are specified, driver_node_type_id and node_type_id take precedence.
    """

    enable_elastic_disk: VariableOrOptional[bool] = None
    """
    Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk
    space when its Spark workers are running low on disk space. This feature requires specific AWS
    permissions to function correctly - refer to the User Guide for more details.
    """

    enable_local_disk_encryption: VariableOrOptional[bool] = None
    """
    Whether to enable LUKS on cluster VMs' local disks
    """

    gcp_attributes: VariableOrOptional[GcpAttributes] = None
    """
    Attributes related to clusters running on Google Cloud Platform.
    If not specified at cluster creation, a set of default values will be used.
    """

    init_scripts: VariableOrList[InitScriptInfo] = field(default_factory=list)
    """
    The configuration for storing init scripts. Any number of destinations can be specified.
    The scripts are executed sequentially in the order provided.
    If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
    """

    instance_pool_id: VariableOrOptional[str] = None
    """
    The optional ID of the instance pool to which the cluster belongs.
    """

    is_single_node: VariableOrOptional[bool] = None
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    When set to true, Databricks will automatically set single node related `custom_tags`, `spark_conf`, and `num_workers`
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

    runtime_engine: VariableOrOptional[RuntimeEngine] = None

    single_user_name: VariableOrOptional[str] = None
    """
    Single user name if data_security_mode is `SINGLE_USER`
    """

    spark_conf: VariableOrDict[str] = field(default_factory=dict)
    """
    An object containing a set of optional, user-specified Spark configuration key-value pairs.
    Users can also pass in a string of extra JVM options to the driver and the executors via
    `spark.driver.extraJavaOptions` and `spark.executor.extraJavaOptions` respectively.
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

    spark_version: VariableOrOptional[str] = None
    """
    The Spark version of the cluster, e.g. `3.3.x-scala2.11`.
    A list of available Spark versions can be retrieved by using
    the :method:clusters/sparkVersions API call.
    """

    ssh_public_keys: VariableOrList[str] = field(default_factory=list)
    """
    SSH public key contents that will be added to each Spark node in this cluster. The
    corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
    Up to 10 keys can be specified.
    """

    use_ml_runtime: VariableOrOptional[bool] = None
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    `effective_spark_version` is determined by `spark_version` (DBR release), this field `use_ml_runtime`, and whether `node_type_id` is gpu node or not.
    """

    workload_type: VariableOrOptional[WorkloadType] = None

    @classmethod
    def from_dict(cls, value: "ClusterSpecDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ClusterSpecDict":
        return _transform_to_json_value(self)  # type:ignore


class ClusterSpecDict(TypedDict, total=False):
    """"""

    apply_policy_default_values: VariableOrOptional[bool]
    """
    When set to true, fixed and default values from the policy will be used for fields that are omitted. When set to false, only fixed values from the policy will be applied.
    """

    autoscale: VariableOrOptional[AutoScaleParam]
    """
    Parameters needed in order to automatically scale clusters up and down based on load.
    Note: autoscaling works best with DB runtime versions 3.0 or later.
    """

    autotermination_minutes: VariableOrOptional[int]
    """
    Automatically terminates the cluster after it is inactive for this time in minutes. If not set,
    this cluster will not be automatically terminated. If specified, the threshold must be between
    10 and 10000 minutes.
    Users can also set this value to 0 to explicitly disable automatic termination.
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
    Three kinds of destinations (DBFS, S3 and Unity Catalog volumes) are supported. Only one destination can be specified
    for one cluster. If the conf is given, the logs will be delivered to the destination every
    `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
    the destination of executor logs is `$destination/$clusterId/executor`.
    """

    cluster_name: VariableOrOptional[str]
    """
    Cluster name requested by the user. This doesn't have to be unique.
    If not specified at creation, the cluster name will be an empty string.
    """

    custom_tags: VariableOrDict[str]
    """
    Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
    instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
    
    - Currently, Databricks allows at most 45 custom tags
    
    - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
    """

    data_security_mode: VariableOrOptional[DataSecurityModeParam]

    docker_image: VariableOrOptional[DockerImageParam]

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
    
    This field, along with node_type_id, should not be set if virtual_cluster_size is set.
    If both driver_node_type_id, node_type_id, and virtual_cluster_size are specified, driver_node_type_id and node_type_id take precedence.
    """

    enable_elastic_disk: VariableOrOptional[bool]
    """
    Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk
    space when its Spark workers are running low on disk space. This feature requires specific AWS
    permissions to function correctly - refer to the User Guide for more details.
    """

    enable_local_disk_encryption: VariableOrOptional[bool]
    """
    Whether to enable LUKS on cluster VMs' local disks
    """

    gcp_attributes: VariableOrOptional[GcpAttributesParam]
    """
    Attributes related to clusters running on Google Cloud Platform.
    If not specified at cluster creation, a set of default values will be used.
    """

    init_scripts: VariableOrList[InitScriptInfoParam]
    """
    The configuration for storing init scripts. Any number of destinations can be specified.
    The scripts are executed sequentially in the order provided.
    If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
    """

    instance_pool_id: VariableOrOptional[str]
    """
    The optional ID of the instance pool to which the cluster belongs.
    """

    is_single_node: VariableOrOptional[bool]
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    When set to true, Databricks will automatically set single node related `custom_tags`, `spark_conf`, and `num_workers`
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

    runtime_engine: VariableOrOptional[RuntimeEngineParam]

    single_user_name: VariableOrOptional[str]
    """
    Single user name if data_security_mode is `SINGLE_USER`
    """

    spark_conf: VariableOrDict[str]
    """
    An object containing a set of optional, user-specified Spark configuration key-value pairs.
    Users can also pass in a string of extra JVM options to the driver and the executors via
    `spark.driver.extraJavaOptions` and `spark.executor.extraJavaOptions` respectively.
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

    spark_version: VariableOrOptional[str]
    """
    The Spark version of the cluster, e.g. `3.3.x-scala2.11`.
    A list of available Spark versions can be retrieved by using
    the :method:clusters/sparkVersions API call.
    """

    ssh_public_keys: VariableOrList[str]
    """
    SSH public key contents that will be added to each Spark node in this cluster. The
    corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
    Up to 10 keys can be specified.
    """

    use_ml_runtime: VariableOrOptional[bool]
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    `effective_spark_version` is determined by `spark_version` (DBR release), this field `use_ml_runtime`, and whether `node_type_id` is gpu node or not.
    """

    workload_type: VariableOrOptional[WorkloadTypeParam]


ClusterSpecParam = ClusterSpecDict | ClusterSpec
