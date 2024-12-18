  
## artifacts
Defines the attributes to build an artifact
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <artifacts-entry-name>
     - Map
     - Item of the `artifacts` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - build
     - String
     - An optional set of non-default build commands that you want to run locally before deployment.  For Python wheel builds, the Databricks CLI assumes that it can find a local install of the Python wheel package to run builds, and it runs the command python setup.py bdist_wheel by default during each bundle deployment.  To specify multiple build commands, separate each command with double-ampersand (&&) characters.
  
   * - executable
     - String
     - The executable type.
  
   * - files
     - Sequence
     - The source files for the artifact, defined as an [_](#artifact_file).
  
   * - path
     - String
     - The location where the built artifact will be saved.
  
   * - type
     - String
     - The type of the artifact. Valid values are `wheel` or `jar`
  
  
## bundle
The attributes of the bundle. See [_](/dev-tools/bundles/settings.md#bundle)
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - cluster_id
     - String
     - The ID of a cluster to use to run the bundle. See [_](/dev-tools/bundles/settings.md#cluster_id).
  
   * - compute_id
     - String
     - 
  
   * - databricks_cli_version
     - String
     - The Databricks CLI version to use for the bundle. See [_](/dev-tools/bundles/settings.md#databricks_cli_version).
  
   * - deployment
     - Map
     - The definition of the bundle deployment. For supported attributes, see [_](#deployment) and [_](/dev-tools/bundles/deployment-modes.md).
  
   * - git
     - Map
     - The Git version control details that are associated with your bundle. For supported attributes, see [_](#git) and [_](/dev-tools/bundles/settings.md#git).
  
   * - name
     - String
     - The name of the bundle.
  
   * - uuid
     - String
     - 
  
  
### bundle.deployment
The definition of the bundle deployment
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - fail_on_active_runs
     - Boolean
     - Whether to fail on active runs. If this is set to true a deployment that is running can be interrupted.
  
   * - lock
     - Map
     - The deployment lock attributes. See [_](#lock).
  
  
### bundle.deployment.lock
The deployment lock attributes.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - Whether this lock is enabled.
  
   * - force
     - Boolean
     - Whether to force this lock if it is enabled.
  
  
### bundle.git
The Git version control details that are associated with your bundle.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - branch
     - String
     - The Git branch name. See [_](/dev-tools/bundles/settings.md#git).
  
   * - origin_url
     - String
     - The origin URL of the repository. See [_](/dev-tools/bundles/settings.md#git).
  
  
## experimental
Defines attributes for experimental features.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - pydabs
     - Map
     - The PyDABs configuration.
  
   * - python_wheel_wrapper
     - Boolean
     - Whether to use a Python wheel wrapper
  
   * - scripts
     - Map
     - The commands to run
  
   * - use_legacy_run_as
     - Boolean
     - Whether to use the legacy run_as behavior
  
  
### experimental.pydabs
The PyDABs configuration.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - Whether or not PyDABs (Private Preview) is enabled
  
   * - import
     - Sequence
     - The PyDABs project to import to discover resources, resource generator and mutators
  
   * - venv_path
     - String
     - The Python virtual environment path
  
  
## include
Specifies a list of path globs that contain configuration files to include within the bundle. See [_](/dev-tools/bundles/settings.md#include)
  
  
## permissions
Defines the permissions to apply to experiments, jobs, pipelines, and models defined in the bundle. See [_](/dev-tools/bundles/settings.md#permissions) and [_](/dev-tools/bundles/permissions.md).
  
Each item of `permissions` has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - group_name
     - String
     - The name of the group that has the permission set in level.
  
   * - level
     - String
     - The allowed permission for user, group, service principal defined for this permission.
  
   * - service_principal_name
     - String
     - The name of the service principal that has the permission set in level.
  
   * - user_name
     - String
     - The name of the user that has the permission set in level.
  
  
## presets
Defines bundle deployment presets. See [_](/dev-tools/bundles/deployment-modes.md#presets).
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - jobs_max_concurrent_runs
     - Integer
     - The maximum concurrent runs for a job.
  
   * - name_prefix
     - String
     - The prefix for job runs of the bundle.
  
   * - pipelines_development
     - Boolean
     - Whether pipeline deployments should be locked in development mode.
  
   * - source_linked_deployment
     - Boolean
     - Whether to link the deployment to the bundle source.
  
   * - tags
     - Map
     - The tags for the bundle deployment.
  
   * - trigger_pause_status
     - String
     - A pause status to apply to all job triggers and schedules. Valid values are PAUSED or UNPAUSED.
  
  
## resources
Specifies information about the Databricks resources used by the bundle. See [_](/dev-tools/bundles/resources.md).
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - clusters
     - Map
     - The cluster definitions for the bundle. See [_](/dev-tools/bundles/resources.md#cluster)
  
   * - dashboards
     - Map
     - The dashboard definitions for the bundle. See [_](/dev-tools/bundles/resources.md#dashboard)
  
   * - experiments
     - Map
     - The experiment definitions for the bundle. See [_](/dev-tools/bundles/resources.md#experiment)
  
   * - jobs
     - Map
     - The job definitions for the bundle. See [_](/dev-tools/bundles/resources.md#job)
  
   * - model_serving_endpoints
     - Map
     - The model serving endpoint definitions for the bundle. See [_](/dev-tools/bundles/resources.md#model_serving_endpoint)
  
   * - models
     - Map
     - The model definitions for the bundle. See [_](/dev-tools/bundles/resources.md#model)
  
   * - pipelines
     - Map
     - The pipeline definitions for the bundle. See [_](/dev-tools/bundles/resources.md#pipeline)
  
   * - quality_monitors
     - Map
     - The quality monitor definitions for the bundle. See [_](/dev-tools/bundles/resources.md#quality_monitor)
  
   * - registered_models
     - Map
     - The registered model definitions for the bundle. See [_](/dev-tools/bundles/resources.md#registered_model)
  
   * - schemas
     - Map
     - The schema definitions for the bundle. See [_](/dev-tools/bundles/resources.md#schema)
  
   * - volumes
     - Map
     - 
  
  
### resources.clusters
The cluster definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.clusters-entry-name>
     - Map
     - Item of the `resources.clusters` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - apply_policy_default_values
     - Boolean
     - When set to true, fixed and default values from the policy will be used for fields that are omitted. When set to false, only fixed values from the policy will be applied.
  
   * - autoscale
     - Map
     - Parameters needed in order to automatically scale clusters up and down based on load. Note: autoscaling works best with DB runtime versions 3.0 or later.
  
   * - autotermination_minutes
     - Integer
     - Automatically terminates the cluster after it is inactive for this time in minutes. If not set, this cluster will not be automatically terminated. If specified, the threshold must be between 10 and 10000 minutes. Users can also set this value to 0 to explicitly disable automatic termination.
  
   * - aws_attributes
     - Map
     - Attributes related to clusters running on Amazon Web Services. If not specified at cluster creation, a set of default values will be used.
  
   * - azure_attributes
     - Map
     - Attributes related to clusters running on Microsoft Azure. If not specified at cluster creation, a set of default values will be used.
  
   * - cluster_log_conf
     - Map
     - The configuration for delivering spark logs to a long-term storage destination. Two kinds of destinations (dbfs and s3) are supported. Only one destination can be specified for one cluster. If the conf is given, the logs will be delivered to the destination every `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while the destination of executor logs is `$destination/$clusterId/executor`.
  
   * - cluster_name
     - String
     - Cluster name requested by the user. This doesn't have to be unique. If not specified at creation, the cluster name will be an empty string. 
  
   * - custom_tags
     - Map
     - Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS instances and EBS volumes) with these tags in addition to `default_tags`. Notes:  - Currently, Databricks allows at most 45 custom tags  - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
  
   * - data_security_mode
     - String
     - 
  
   * - docker_image
     - Map
     - 
  
   * - driver_instance_pool_id
     - String
     - The optional ID of the instance pool for the driver of the cluster belongs. The pool cluster uses the instance pool with id (instance_pool_id) if the driver pool is not assigned.
  
   * - driver_node_type_id
     - String
     - The node type of the Spark driver. Note that this field is optional; if unset, the driver node type will be set as the same value as `node_type_id` defined above. 
  
   * - enable_elastic_disk
     - Boolean
     - Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space. This feature requires specific AWS permissions to function correctly - refer to the User Guide for more details.
  
   * - enable_local_disk_encryption
     - Boolean
     - Whether to enable LUKS on cluster VMs' local disks
  
   * - gcp_attributes
     - Map
     - Attributes related to clusters running on Google Cloud Platform. If not specified at cluster creation, a set of default values will be used.
  
   * - init_scripts
     - Sequence
     - The configuration for storing init scripts. Any number of destinations can be specified. The scripts are executed sequentially in the order provided. If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
  
   * - instance_pool_id
     - String
     - The optional ID of the instance pool to which the cluster belongs.
  
   * - node_type_id
     - String
     - This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster. For example, the Spark nodes can be provisioned and optimized for memory or compute intensive workloads. A list of available node types can be retrieved by using the :method:clusters/listNodeTypes API call. 
  
   * - num_workers
     - Integer
     - Number of worker nodes that this cluster should have. A cluster has one Spark Driver and `num_workers` Executors for a total of `num_workers` + 1 Spark nodes.  Note: When reading the properties of a cluster, this field reflects the desired number of workers rather than the actual current number of workers. For instance, if a cluster is resized from 5 to 10 workers, this field will immediately be updated to reflect the target size of 10 workers, whereas the workers listed in `spark_info` will gradually increase from 5 to 10 as the new nodes are provisioned.
  
   * - permissions
     - Sequence
     - 
  
   * - policy_id
     - String
     - The ID of the cluster policy used to create the cluster if applicable.
  
   * - runtime_engine
     - String
     - 
  
   * - single_user_name
     - String
     - Single user name if data_security_mode is `SINGLE_USER`
  
   * - spark_conf
     - Map
     - An object containing a set of optional, user-specified Spark configuration key-value pairs. Users can also pass in a string of extra JVM options to the driver and the executors via `spark.driver.extraJavaOptions` and `spark.executor.extraJavaOptions` respectively. 
  
   * - spark_env_vars
     - Map
     - An object containing a set of optional, user-specified environment variable key-value pairs. Please note that key-value pair of the form (X,Y) will be exported as is (i.e., `export X='Y'`) while launching the driver and workers.  In order to specify an additional set of `SPARK_DAEMON_JAVA_OPTS`, we recommend appending them to `$SPARK_DAEMON_JAVA_OPTS` as shown in the example below. This ensures that all default databricks managed environmental variables are included as well.  Example Spark environment variables: `{"SPARK_WORKER_MEMORY": "28000m", "SPARK_LOCAL_DIRS": "/local_disk0"}` or `{"SPARK_DAEMON_JAVA_OPTS": "$SPARK_DAEMON_JAVA_OPTS -Dspark.shuffle.service.enabled=true"}`
  
   * - spark_version
     - String
     - The Spark version of the cluster, e.g. `3.3.x-scala2.11`. A list of available Spark versions can be retrieved by using the :method:clusters/sparkVersions API call. 
  
   * - ssh_public_keys
     - Sequence
     - SSH public key contents that will be added to each Spark node in this cluster. The corresponding private keys can be used to login with the user name `ubuntu` on port `2200`. Up to 10 keys can be specified.
  
   * - workload_type
     - Map
     - 
  
  
### resources.clusters.autoscale
Parameters needed in order to automatically scale clusters up and down based on load.
Note: autoscaling works best with DB runtime versions 3.0 or later.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - max_workers
     - Integer
     - The maximum number of workers to which the cluster can scale up when overloaded. Note that `max_workers` must be strictly greater than `min_workers`.
  
   * - min_workers
     - Integer
     - The minimum number of workers to which the cluster can scale down when underutilized. It is also the initial number of workers the cluster will have after creation.
  
  
### resources.clusters.aws_attributes
Attributes related to clusters running on Amazon Web Services.
If not specified at cluster creation, a set of default values will be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - availability
     - String
     - 
  
   * - ebs_volume_count
     - Integer
     - The number of volumes launched for each instance. Users can choose up to 10 volumes. This feature is only enabled for supported node types. Legacy node types cannot specify custom EBS volumes. For node types with no instance store, at least one EBS volume needs to be specified; otherwise, cluster creation will fail.  These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc. Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.  If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for scratch storage because heterogenously sized scratch devices can lead to inefficient disk utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance store volumes.  Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir` will be overridden.
  
   * - ebs_volume_iops
     - Integer
     - If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
  
   * - ebs_volume_size
     - Integer
     - The size of each EBS volume (in GiB) launched for each instance. For general purpose SSD, this value must be within the range 100 - 4096. For throughput optimized HDD, this value must be within the range 500 - 4096.
  
   * - ebs_volume_throughput
     - Integer
     - If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
  
   * - ebs_volume_type
     - String
     - 
  
   * - first_on_demand
     - Integer
     - The first `first_on_demand` nodes of the cluster will be placed on on-demand instances. If this value is greater than 0, the cluster driver node in particular will be placed on an on-demand instance. If this value is greater than or equal to the current cluster size, all nodes will be placed on on-demand instances. If this value is less than the current cluster size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will be placed on `availability` instances. Note that this value does not affect cluster size and cannot currently be mutated over the lifetime of a cluster.
  
   * - instance_profile_arn
     - String
     - Nodes for this cluster will only be placed on AWS instances with this instance profile. If ommitted, nodes will be placed on instances without an IAM instance profile. The instance profile must have previously been added to the Databricks environment by an account administrator.  This feature may only be available to certain customer plans.  If this field is ommitted, we will pull in the default from the conf if it exists.
  
   * - spot_bid_price_percent
     - Integer
     - The bid price for AWS spot instances, as a percentage of the corresponding instance type's on-demand price. For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot instance, then the bid price is half of the price of on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100. When spot instances are requested for this cluster, only spot instances whose bid price percentage matches this field will be considered. Note that, for safety, we enforce this field to be no more than 10000.  The default value and documentation here should be kept consistent with CommonConf.defaultSpotBidPricePercent and CommonConf.maxSpotBidPricePercent.
  
   * - zone_id
     - String
     - Identifier for the availability zone/datacenter in which the cluster resides. This string will be of a form like "us-west-2a". The provided availability zone must be in the same region as the Databricks deployment. For example, "us-west-2a" is not a valid zone id if the Databricks deployment resides in the "us-east-1" region. This is an optional field at cluster creation, and if not specified, a default zone will be used. If the zone specified is "auto", will try to place cluster in a zone with high availability, and will retry placement in a different AZ if there is not enough capacity. The list of available zones as well as the default value can be found by using the `List Zones` method.
  
  
### resources.clusters.azure_attributes
Attributes related to clusters running on Microsoft Azure.
If not specified at cluster creation, a set of default values will be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - availability
     - String
     - 
  
   * - first_on_demand
     - Integer
     - The first `first_on_demand` nodes of the cluster will be placed on on-demand instances. This value should be greater than 0, to make sure the cluster driver node is placed on an on-demand instance. If this value is greater than or equal to the current cluster size, all nodes will be placed on on-demand instances. If this value is less than the current cluster size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will be placed on `availability` instances. Note that this value does not affect cluster size and cannot currently be mutated over the lifetime of a cluster.
  
   * - log_analytics_info
     - Map
     - Defines values necessary to configure and run Azure Log Analytics agent
  
   * - spot_bid_max_price
     - Any
     - The max bid price to be used for Azure spot instances. The Max price for the bid cannot be higher than the on-demand price of the instance. If not specified, the default value is -1, which specifies that the instance cannot be evicted on the basis of price, and only on the basis of availability. Further, the value should > 0 or -1.
  
  
### resources.clusters.azure_attributes.log_analytics_info
Defines values necessary to configure and run Azure Log Analytics agent
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - log_analytics_primary_key
     - String
     - <needs content added>
  
   * - log_analytics_workspace_id
     - String
     - <needs content added>
  
  
### resources.clusters.cluster_log_conf
The configuration for delivering spark logs to a long-term storage destination.
Two kinds of destinations (dbfs and s3) are supported. Only one destination can be specified
for one cluster. If the conf is given, the logs will be delivered to the destination every
`5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
the destination of executor logs is `$destination/$clusterId/executor`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - dbfs
     - Map
     - destination needs to be provided. e.g. `{ "dbfs" : { "destination" : "dbfs:/home/cluster_log" } }`
  
   * - s3
     - Map
     - destination and either the region or endpoint need to be provided. e.g. `{ "s3": { "destination" : "s3://cluster_log_bucket/prefix", "region" : "us-west-2" } }` Cluster iam role is used to access s3, please make sure the cluster iam role in `instance_profile_arn` has permission to write data to the s3 destination.
  
  
### resources.clusters.cluster_log_conf.dbfs
destination needs to be provided. e.g.
`{ "dbfs" : { "destination" : "dbfs:/home/cluster_log" } }`
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - destination
     - String
     - dbfs destination, e.g. `dbfs:/my/path`
  
  
### resources.clusters.cluster_log_conf.s3
destination and either the region or endpoint need to be provided. e.g.
`{ "s3": { "destination" : "s3://cluster_log_bucket/prefix", "region" : "us-west-2" } }`
Cluster iam role is used to access s3, please make sure the cluster iam role in
`instance_profile_arn` has permission to write data to the s3 destination.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - canned_acl
     - String
     - (Optional) Set canned access control list for the logs, e.g. `bucket-owner-full-control`. If `canned_cal` is set, please make sure the cluster iam role has `s3:PutObjectAcl` permission on the destination bucket and prefix. The full list of possible canned acl can be found at http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl. Please also note that by default only the object owner gets full controls. If you are using cross account role for writing data, you may want to set `bucket-owner-full-control` to make bucket owner able to read the logs.
  
   * - destination
     - String
     - S3 destination, e.g. `s3://my-bucket/some-prefix` Note that logs will be delivered using cluster iam role, please make sure you set cluster iam role and the role has write access to the destination. Please also note that you cannot use AWS keys to deliver logs.
  
   * - enable_encryption
     - Boolean
     - (Optional) Flag to enable server side encryption, `false` by default.
  
   * - encryption_type
     - String
     - (Optional) The encryption type, it could be `sse-s3` or `sse-kms`. It will be used only when encryption is enabled and the default type is `sse-s3`.
  
   * - endpoint
     - String
     - S3 endpoint, e.g. `https://s3-us-west-2.amazonaws.com`. Either region or endpoint needs to be set. If both are set, endpoint will be used.
  
   * - kms_key
     - String
     - (Optional) Kms key which will be used if encryption is enabled and encryption type is set to `sse-kms`.
  
   * - region
     - String
     - S3 region, e.g. `us-west-2`. Either region or endpoint needs to be set. If both are set, endpoint will be used.
  
  
### resources.clusters.docker_image

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - basic_auth
     - Map
     - 
  
   * - url
     - String
     - URL of the docker image.
  
  
### resources.clusters.docker_image.basic_auth

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - password
     - String
     - Password of the user
  
   * - username
     - String
     - Name of the user
  
  
### resources.clusters.gcp_attributes
Attributes related to clusters running on Google Cloud Platform.
If not specified at cluster creation, a set of default values will be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - availability
     - String
     - 
  
   * - boot_disk_size
     - Integer
     - boot disk size in GB
  
   * - google_service_account
     - String
     - If provided, the cluster will impersonate the google service account when accessing gcloud services (like GCS). The google service account must have previously been added to the Databricks environment by an account administrator.
  
   * - local_ssd_count
     - Integer
     - If provided, each node (workers and driver) in the cluster will have this number of local SSDs attached. Each local SSD is 375GB in size. Refer to [GCP documentation](https://cloud.google.com/compute/docs/disks/local-ssd#choose_number_local_ssds) for the supported number of local SSDs for each instance type.
  
   * - use_preemptible_executors
     - Boolean
     - This field determines whether the spark executors will be scheduled to run on preemptible VMs (when set to true) versus standard compute engine VMs (when set to false; default). Note: Soon to be deprecated, use the availability field instead.
  
   * - zone_id
     - String
     - Identifier for the availability zone in which the cluster resides. This can be one of the following: - "HA" => High availability, spread nodes across availability zones for a Databricks deployment region [default] - "AUTO" => Databricks picks an availability zone to schedule the cluster on. - A GCP availability zone => Pick One of the available zones for (machine type + region) from https://cloud.google.com/compute/docs/regions-zones.
  
  
### resources.clusters.workload_type

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - clients
     - Map
     -  defined what type of clients can use the cluster. E.g. Notebooks, Jobs
  
  
### resources.clusters.workload_type.clients
 defined what type of clients can use the cluster. E.g. Notebooks, Jobs
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - jobs
     - Boolean
     - With jobs set, the cluster can be used for jobs
  
   * - notebooks
     - Boolean
     - With notebooks set, this cluster can be used for notebooks
  
  
### resources.dashboards
The dashboard definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.dashboards-entry-name>
     - Map
     - Item of the `resources.dashboards` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - create_time
     - String
     - The timestamp of when the dashboard was created.
  
   * - dashboard_id
     - String
     - UUID identifying the dashboard.
  
   * - display_name
     - String
     - The display name of the dashboard.
  
   * - embed_credentials
     - Boolean
     - 
  
   * - etag
     - String
     - The etag for the dashboard. Can be optionally provided on updates to ensure that the dashboard has not been modified since the last read. This field is excluded in List Dashboards responses.
  
   * - file_path
     - String
     - 
  
   * - lifecycle_state
     - String
     - The state of the dashboard resource. Used for tracking trashed status.
  
   * - parent_path
     - String
     - The workspace path of the folder containing the dashboard. Includes leading slash and no trailing slash. This field is excluded in List Dashboards responses.
  
   * - path
     - String
     - The workspace path of the dashboard asset, including the file name. Exported dashboards always have the file extension `.lvdash.json`. This field is excluded in List Dashboards responses.
  
   * - permissions
     - Sequence
     - 
  
   * - serialized_dashboard
     - Any
     - The contents of the dashboard in serialized string form. This field is excluded in List Dashboards responses. Use the [get dashboard API](https://docs.databricks.com/api/workspace/lakeview/get) to retrieve an example response, which includes the `serialized_dashboard` field. This field provides the structure of the JSON string that represents the dashboard's layout and components.
  
   * - update_time
     - String
     - The timestamp of when the dashboard was last updated by the user. This field is excluded in List Dashboards responses.
  
   * - warehouse_id
     - String
     - The warehouse ID used to run the dashboard.
  
  
### resources.experiments
The experiment definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.experiments-entry-name>
     - Map
     - Item of the `resources.experiments` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - artifact_location
     - String
     - Location where artifacts for the experiment are stored.
  
   * - creation_time
     - Integer
     - Creation time
  
   * - experiment_id
     - String
     - Unique identifier for the experiment.
  
   * - last_update_time
     - Integer
     - Last update time
  
   * - lifecycle_stage
     - String
     - Current life cycle stage of the experiment: "active" or "deleted". Deleted experiments are not returned by APIs.
  
   * - name
     - String
     - Human readable name that identifies the experiment.
  
   * - permissions
     - Sequence
     - 
  
   * - tags
     - Sequence
     - Tags: Additional metadata key-value pairs.
  
  
### resources.jobs
The job definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.jobs-entry-name>
     - Map
     - Item of the `resources.jobs` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - budget_policy_id
     - String
     - The id of the user specified budget policy to use for this job. If not specified, a default budget policy may be applied when creating or modifying the job. See `effective_budget_policy_id` for the budget policy used by this workload.
  
   * - continuous
     - Map
     - An optional continuous property for this job. The continuous property will ensure that there is always one run executing. Only one of `schedule` and `continuous` can be used.
  
   * - description
     - String
     - An optional description for the job. The maximum length is 27700 characters in UTF-8 encoding.
  
   * - email_notifications
     - Map
     - An optional set of email addresses that is notified when runs of this job begin or complete as well as when this job is deleted.
  
   * - environments
     - Sequence
     - A list of task execution environment specifications that can be referenced by serverless tasks of this job. An environment is required to be present for serverless tasks. For serverless notebook tasks, the environment is accessible in the notebook environment panel. For other serverless tasks, the task environment is required to be specified using environment_key in the task settings.
  
   * - git_source
     - Map
     - An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.  If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.  Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
  
   * - health
     - Map
     - 
  
   * - job_clusters
     - Sequence
     - A list of job cluster specifications that can be shared and reused by tasks of this job. Libraries cannot be declared in a shared job cluster. You must declare dependent libraries in task settings.
  
   * - max_concurrent_runs
     - Integer
     - An optional maximum allowed number of concurrent runs of the job. Set this value if you want to be able to execute multiple runs of the same job concurrently. This is useful for example if you trigger your job on a frequent schedule and want to allow consecutive runs to overlap with each other, or if you want to trigger multiple runs which differ by their input parameters. This setting affects only new runs. For example, suppose the job’s concurrency is 4 and there are 4 concurrent active runs. Then setting the concurrency to 3 won’t kill any of the active runs. However, from then on, new runs are skipped unless there are fewer than 3 active runs. This value cannot exceed 1000. Setting this value to `0` causes all new runs to be skipped.
  
   * - name
     - String
     - An optional name for the job. The maximum length is 4096 bytes in UTF-8 encoding.
  
   * - notification_settings
     - Map
     - Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this job.
  
   * - parameters
     - Sequence
     - Job-level parameter definitions
  
   * - permissions
     - Sequence
     - 
  
   * - queue
     - Map
     - The queue settings of the job.
  
   * - run_as
     - Map
     - 
  
   * - schedule
     - Map
     - An optional periodic schedule for this job. The default behavior is that the job only runs when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
   * - tags
     - Map
     - A map of tags associated with the job. These are forwarded to the cluster as cluster tags for jobs clusters, and are subject to the same limitations as cluster tags. A maximum of 25 tags can be added to the job.
  
   * - tasks
     - Sequence
     - A list of task specifications to be executed by this job.
  
   * - timeout_seconds
     - Integer
     - An optional timeout applied to each run of this job. A value of `0` means no timeout.
  
   * - trigger
     - Map
     - A configuration to trigger a run when certain conditions are met. The default behavior is that the job runs only when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
   * - webhook_notifications
     - Map
     - A collection of system notification IDs to notify when runs of this job begin or complete.
  
  
### resources.jobs.continuous
An optional continuous property for this job. The continuous property will ensure that there is always one run executing. Only one of `schedule` and `continuous` can be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - pause_status
     - String
     - Indicate whether the continuous execution of the job is paused or not. Defaults to UNPAUSED.
  
  
### resources.jobs.email_notifications
An optional set of email addresses that is notified when runs of this job begin or complete as well as when this job is deleted.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - no_alert_for_skipped_runs
     - Boolean
     - If true, do not send email to recipients specified in `on_failure` if the run is skipped. This field is `deprecated`. Please use the `notification_settings.no_alert_for_skipped_runs` field.
  
   * - on_duration_warning_threshold_exceeded
     - Sequence
     - A list of email addresses to be notified when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. If no rule for the `RUN_DURATION_SECONDS` metric is specified in the `health` field for the job, notifications are not sent.
  
   * - on_failure
     - Sequence
     - A list of email addresses to be notified when a run unsuccessfully completes. A run is considered to have completed unsuccessfully if it ends with an `INTERNAL_ERROR` `life_cycle_state` or a `FAILED`, or `TIMED_OUT` result_state. If this is not specified on job creation, reset, or update the list is empty, and notifications are not sent.
  
   * - on_start
     - Sequence
     - A list of email addresses to be notified when a run begins. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
  
   * - on_streaming_backlog_exceeded
     - Sequence
     - A list of email addresses to notify when any streaming backlog thresholds are exceeded for any stream. Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`. Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
  
   * - on_success
     - Sequence
     - A list of email addresses to be notified when a run successfully completes. A run is considered to have completed successfully if it ends with a `TERMINATED` `life_cycle_state` and a `SUCCESS` result_state. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
  
  
### resources.jobs.git_source
An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.

If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.

Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - git_branch
     - String
     - Name of the branch to be checked out and used by this job. This field cannot be specified in conjunction with git_tag or git_commit.
  
   * - git_commit
     - String
     - Commit to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_tag.
  
   * - git_provider
     - String
     - Unique identifier of the service used to host the Git repository. The value is case insensitive.
  
   * - git_tag
     - String
     - Name of the tag to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_commit.
  
   * - git_url
     - String
     - URL of the repository to be cloned by this job.
  
  
### resources.jobs.health

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - rules
     - Sequence
     - 
  
  
### resources.jobs.notification_settings
Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this job.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - no_alert_for_canceled_runs
     - Boolean
     - If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
  
   * - no_alert_for_skipped_runs
     - Boolean
     - If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
  
  
### resources.jobs.queue
The queue settings of the job.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - If true, enable queueing for the job. This is a required field.
  
  
### resources.jobs.run_as

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - service_principal_name
     - String
     - Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
  
   * - user_name
     - String
     - The email of an active workspace user. Non-admin users can only set this field to their own email.
  
  
### resources.jobs.schedule
An optional periodic schedule for this job. The default behavior is that the job only runs when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - pause_status
     - String
     - Indicate whether this schedule is paused or not.
  
   * - quartz_cron_expression
     - String
     - A Cron expression using Quartz syntax that describes the schedule for a job. See [Cron Trigger](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html) for details. This field is required.
  
   * - timezone_id
     - String
     - A Java timezone ID. The schedule for a job is resolved with respect to this timezone. See [Java TimeZone](https://docs.oracle.com/javase/7/docs/api/java/util/TimeZone.html) for details. This field is required.
  
  
### resources.jobs.trigger
A configuration to trigger a run when certain conditions are met. The default behavior is that the job runs only when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - file_arrival
     - Map
     - File arrival trigger settings.
  
   * - pause_status
     - String
     - Whether this trigger is paused or not.
  
   * - periodic
     - Map
     - Periodic trigger settings.
  
   * - table
     - Map
     - Old table trigger settings name. Deprecated in favor of `table_update`.
  
   * - table_update
     - Map
     - 
  
  
### resources.jobs.trigger.file_arrival
File arrival trigger settings.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - min_time_between_triggers_seconds
     - Integer
     - If set, the trigger starts a run only after the specified amount of time passed since the last time the trigger fired. The minimum allowed value is 60 seconds
  
   * - url
     - String
     - URL to be monitored for file arrivals. The path must point to the root or a subpath of the external location.
  
   * - wait_after_last_change_seconds
     - Integer
     - If set, the trigger starts a run only after no file activity has occurred for the specified amount of time. This makes it possible to wait for a batch of incoming files to arrive before triggering a run. The minimum allowed value is 60 seconds.
  
  
### resources.jobs.trigger.periodic
Periodic trigger settings.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - interval
     - Integer
     - The interval at which the trigger should run.
  
   * - unit
     - String
     - The unit of time for the interval.
  
  
### resources.jobs.trigger.table
Old table trigger settings name. Deprecated in favor of `table_update`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - condition
     - String
     - The table(s) condition based on which to trigger a job run.
  
   * - min_time_between_triggers_seconds
     - Integer
     - If set, the trigger starts a run only after the specified amount of time has passed since the last time the trigger fired. The minimum allowed value is 60 seconds.
  
   * - table_names
     - Sequence
     - A list of Delta tables to monitor for changes. The table name must be in the format `catalog_name.schema_name.table_name`.
  
   * - wait_after_last_change_seconds
     - Integer
     - If set, the trigger starts a run only after no table updates have occurred for the specified time and can be used to wait for a series of table updates before triggering a run. The minimum allowed value is 60 seconds.
  
  
### resources.jobs.trigger.table_update

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - condition
     - String
     - The table(s) condition based on which to trigger a job run.
  
   * - min_time_between_triggers_seconds
     - Integer
     - If set, the trigger starts a run only after the specified amount of time has passed since the last time the trigger fired. The minimum allowed value is 60 seconds.
  
   * - table_names
     - Sequence
     - A list of Delta tables to monitor for changes. The table name must be in the format `catalog_name.schema_name.table_name`.
  
   * - wait_after_last_change_seconds
     - Integer
     - If set, the trigger starts a run only after no table updates have occurred for the specified time and can be used to wait for a series of table updates before triggering a run. The minimum allowed value is 60 seconds.
  
  
### resources.jobs.webhook_notifications
A collection of system notification IDs to notify when runs of this job begin or complete.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - on_duration_warning_threshold_exceeded
     - Sequence
     - An optional list of system notification IDs to call when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. A maximum of 3 destinations can be specified for the `on_duration_warning_threshold_exceeded` property.
  
   * - on_failure
     - Sequence
     - An optional list of system notification IDs to call when the run fails. A maximum of 3 destinations can be specified for the `on_failure` property.
  
   * - on_start
     - Sequence
     - An optional list of system notification IDs to call when the run starts. A maximum of 3 destinations can be specified for the `on_start` property.
  
   * - on_streaming_backlog_exceeded
     - Sequence
     - An optional list of system notification IDs to call when any streaming backlog thresholds are exceeded for any stream. Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`. Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes. A maximum of 3 destinations can be specified for the `on_streaming_backlog_exceeded` property.
  
   * - on_success
     - Sequence
     - An optional list of system notification IDs to call when the run completes successfully. A maximum of 3 destinations can be specified for the `on_success` property.
  
  
### resources.model_serving_endpoints
The model serving endpoint definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.model_serving_endpoints-entry-name>
     - Map
     - Item of the `resources.model_serving_endpoints` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - ai_gateway
     - Map
     - The AI Gateway configuration for the serving endpoint. NOTE: only external model endpoints are supported as of now.
  
   * - config
     - Map
     - The core config of the serving endpoint.
  
   * - name
     - String
     - The name of the serving endpoint. This field is required and must be unique across a Databricks workspace. An endpoint name can consist of alphanumeric characters, dashes, and underscores. 
  
   * - permissions
     - Sequence
     - 
  
   * - rate_limits
     - Sequence
     - Rate limits to be applied to the serving endpoint. NOTE: this field is deprecated, please use AI Gateway to manage rate limits.
  
   * - route_optimized
     - Boolean
     - Enable route optimization for the serving endpoint.
  
   * - tags
     - Sequence
     - Tags to be attached to the serving endpoint and automatically propagated to billing logs.
  
  
### resources.model_serving_endpoints.ai_gateway
The AI Gateway configuration for the serving endpoint. NOTE: only external model endpoints are supported as of now.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - guardrails
     - Map
     - Configuration for AI Guardrails to prevent unwanted data and unsafe data in requests and responses.
  
   * - inference_table_config
     - Map
     - Configuration for payload logging using inference tables. Use these tables to monitor and audit data being sent to and received from model APIs and to improve model quality.
  
   * - rate_limits
     - Sequence
     - Configuration for rate limits which can be set to limit endpoint traffic.
  
   * - usage_tracking_config
     - Map
     - Configuration to enable usage tracking using system tables. These tables allow you to monitor operational usage on endpoints and their associated costs.
  
  
### resources.model_serving_endpoints.ai_gateway.guardrails
Configuration for AI Guardrails to prevent unwanted data and unsafe data in requests and responses.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - input
     - Map
     - Configuration for input guardrail filters.
  
   * - output
     - Map
     - Configuration for output guardrail filters.
  
  
### resources.model_serving_endpoints.ai_gateway.guardrails.input
Configuration for input guardrail filters.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - invalid_keywords
     - Sequence
     - List of invalid keywords. AI guardrail uses keyword or string matching to decide if the keyword exists in the request or response content.
  
   * - pii
     - Map
     - Configuration for guardrail PII filter.
  
   * - safety
     - Boolean
     - Indicates whether the safety filter is enabled.
  
   * - valid_topics
     - Sequence
     - The list of allowed topics. Given a chat request, this guardrail flags the request if its topic is not in the allowed topics.
  
  
### resources.model_serving_endpoints.ai_gateway.guardrails.input.pii
Configuration for guardrail PII filter.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - behavior
     - String
     - Behavior for PII filter. Currently only 'BLOCK' is supported. If 'BLOCK' is set for the input guardrail and the request contains PII, the request is not sent to the model server and 400 status code is returned; if 'BLOCK' is set for the output guardrail and the model response contains PII, the PII info in the response is redacted and 400 status code is returned.
  
  
### resources.model_serving_endpoints.ai_gateway.guardrails.output
Configuration for output guardrail filters.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - invalid_keywords
     - Sequence
     - List of invalid keywords. AI guardrail uses keyword or string matching to decide if the keyword exists in the request or response content.
  
   * - pii
     - Map
     - Configuration for guardrail PII filter.
  
   * - safety
     - Boolean
     - Indicates whether the safety filter is enabled.
  
   * - valid_topics
     - Sequence
     - The list of allowed topics. Given a chat request, this guardrail flags the request if its topic is not in the allowed topics.
  
  
### resources.model_serving_endpoints.ai_gateway.guardrails.output.pii
Configuration for guardrail PII filter.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - behavior
     - String
     - Behavior for PII filter. Currently only 'BLOCK' is supported. If 'BLOCK' is set for the input guardrail and the request contains PII, the request is not sent to the model server and 400 status code is returned; if 'BLOCK' is set for the output guardrail and the model response contains PII, the PII info in the response is redacted and 400 status code is returned.
  
  
### resources.model_serving_endpoints.ai_gateway.inference_table_config
Configuration for payload logging using inference tables. Use these tables to monitor and audit data being sent to and received from model APIs and to improve model quality.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog in Unity Catalog. Required when enabling inference tables. NOTE: On update, you have to disable inference table first in order to change the catalog name.
  
   * - enabled
     - Boolean
     - Indicates whether the inference table is enabled.
  
   * - schema_name
     - String
     - The name of the schema in Unity Catalog. Required when enabling inference tables. NOTE: On update, you have to disable inference table first in order to change the schema name.
  
   * - table_name_prefix
     - String
     - The prefix of the table in Unity Catalog. NOTE: On update, you have to disable inference table first in order to change the prefix name.
  
  
### resources.model_serving_endpoints.ai_gateway.usage_tracking_config
Configuration to enable usage tracking using system tables. These tables allow you to monitor operational usage on endpoints and their associated costs.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - Whether to enable usage tracking.
  
  
### resources.model_serving_endpoints.config
The core config of the serving endpoint.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - auto_capture_config
     - Map
     - Configuration for Inference Tables which automatically logs requests and responses to Unity Catalog.
  
   * - served_entities
     - Sequence
     - A list of served entities for the endpoint to serve. A serving endpoint can have up to 15 served entities.
  
   * - served_models
     - Sequence
     - (Deprecated, use served_entities instead) A list of served models for the endpoint to serve. A serving endpoint can have up to 15 served models.
  
   * - traffic_config
     - Map
     - The traffic config defining how invocations to the serving endpoint should be routed.
  
  
### resources.model_serving_endpoints.config.auto_capture_config
Configuration for Inference Tables which automatically logs requests and responses to Unity Catalog.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog in Unity Catalog. NOTE: On update, you cannot change the catalog name if the inference table is already enabled.
  
   * - enabled
     - Boolean
     - Indicates whether the inference table is enabled.
  
   * - schema_name
     - String
     - The name of the schema in Unity Catalog. NOTE: On update, you cannot change the schema name if the inference table is already enabled.
  
   * - table_name_prefix
     - String
     - The prefix of the table in Unity Catalog. NOTE: On update, you cannot change the prefix name if the inference table is already enabled.
  
  
### resources.model_serving_endpoints.config.traffic_config
The traffic config defining how invocations to the serving endpoint should be routed.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - routes
     - Sequence
     - The list of routes that define traffic to each served entity.
  
  
### resources.models
The model definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.models-entry-name>
     - Map
     - Item of the `resources.models` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - creation_timestamp
     - Integer
     - Timestamp recorded when this `registered_model` was created.
  
   * - description
     - String
     - Description of this `registered_model`.
  
   * - last_updated_timestamp
     - Integer
     - Timestamp recorded when metadata for this `registered_model` was last updated.
  
   * - latest_versions
     - Sequence
     - Collection of latest model versions for each stage. Only contains models with current `READY` status.
  
   * - name
     - String
     - Unique name for the model.
  
   * - permissions
     - Sequence
     - 
  
   * - tags
     - Sequence
     - Tags: Additional metadata key-value pairs for this `registered_model`.
  
   * - user_id
     - String
     - User that created this `registered_model`
  
  
### resources.pipelines
The pipeline definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.pipelines-entry-name>
     - Map
     - Item of the `resources.pipelines` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - budget_policy_id
     - String
     - Budget policy of this pipeline.
  
   * - catalog
     - String
     - A catalog in Unity Catalog to publish data from this pipeline to. If `target` is specified, tables in this pipeline are published to a `target` schema inside `catalog` (for example, `catalog`.`target`.`table`). If `target` is not specified, no data is published to Unity Catalog.
  
   * - channel
     - String
     - DLT Release Channel that specifies which version to use.
  
   * - clusters
     - Sequence
     - Cluster settings for this pipeline deployment.
  
   * - configuration
     - Map
     - String-String configuration for this pipeline execution.
  
   * - continuous
     - Boolean
     - Whether the pipeline is continuous or triggered. This replaces `trigger`.
  
   * - deployment
     - Map
     - Deployment type of this pipeline.
  
   * - development
     - Boolean
     - Whether the pipeline is in Development mode. Defaults to false.
  
   * - edition
     - String
     - Pipeline product edition.
  
   * - filters
     - Map
     - Filters on which Pipeline packages to include in the deployed graph.
  
   * - gateway_definition
     - Map
     - The definition of a gateway pipeline to support change data capture.
  
   * - id
     - String
     - Unique identifier for this pipeline.
  
   * - ingestion_definition
     - Map
     - The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'target' or 'catalog' settings.
  
   * - libraries
     - Sequence
     - Libraries or code needed by this deployment.
  
   * - name
     - String
     - Friendly identifier for this pipeline.
  
   * - notifications
     - Sequence
     - List of notification settings for this pipeline.
  
   * - permissions
     - Sequence
     - 
  
   * - photon
     - Boolean
     - Whether Photon is enabled for this pipeline.
  
   * - restart_window
     - Map
     - Restart window of this pipeline.
  
   * - schema
     - String
     - The default schema (database) where tables are read from or published to. The presence of this field implies that the pipeline is in direct publishing mode.
  
   * - serverless
     - Boolean
     - Whether serverless compute is enabled for this pipeline.
  
   * - storage
     - String
     - DBFS root directory for storing checkpoints and tables.
  
   * - target
     - String
     - Target schema (database) to add tables in this pipeline to. If not specified, no data is published to the Hive metastore or Unity Catalog. To publish to Unity Catalog, also specify `catalog`.
  
   * - trigger
     - Map
     - Which pipeline trigger to use. Deprecated: Use `continuous` instead.
  
  
### resources.pipelines.deployment
Deployment type of this pipeline.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - kind
     - String
     - The deployment method that manages the pipeline.
  
   * - metadata_file_path
     - String
     - The path to the file containing metadata about the deployment.
  
  
### resources.pipelines.filters
Filters on which Pipeline packages to include in the deployed graph.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - exclude
     - Sequence
     - Paths to exclude.
  
   * - include
     - Sequence
     - Paths to include.
  
  
### resources.pipelines.gateway_definition
The definition of a gateway pipeline to support change data capture.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - connection_id
     - String
     - [Deprecated, use connection_name instead] Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
  
   * - connection_name
     - String
     - Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
  
   * - gateway_storage_catalog
     - String
     - Required, Immutable. The name of the catalog for the gateway pipeline's storage location.
  
   * - gateway_storage_name
     - String
     - Optional. The Unity Catalog-compatible name for the gateway storage location. This is the destination to use for the data that is extracted by the gateway. Delta Live Tables system will automatically create the storage location under the catalog and schema. 
  
   * - gateway_storage_schema
     - String
     - Required, Immutable. The name of the schema for the gateway pipelines's storage location.
  
  
### resources.pipelines.ingestion_definition
The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'target' or 'catalog' settings.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - connection_name
     - String
     - Immutable. The Unity Catalog connection that this ingestion pipeline uses to communicate with the source. This is used with connectors for applications like Salesforce, Workday, and so on.
  
   * - ingestion_gateway_id
     - String
     - Immutable. Identifier for the gateway that is used by this ingestion pipeline to communicate with the source database. This is used with connectors to databases like SQL Server.
  
   * - objects
     - Sequence
     - Required. Settings specifying tables to replicate and the destination for the replicated tables.
  
   * - table_configuration
     - Map
     - Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
  
  
### resources.pipelines.ingestion_definition.table_configuration
Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - primary_keys
     - Sequence
     - The primary key of the table used to apply changes.
  
   * - salesforce_include_formula_fields
     - Boolean
     - If true, formula fields defined in the table are included in the ingestion. This setting is only valid for the Salesforce connector
  
   * - scd_type
     - String
     - The SCD type to use to ingest the table.
  
   * - sequence_by
     - Sequence
     - The column names specifying the logical order of events in the source data. Delta Live Tables uses this sequencing to handle change events that arrive out of order.
  
  
### resources.pipelines.restart_window
Restart window of this pipeline.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - days_of_week
     - String
     - Days of week in which the restart is allowed to happen (within a five-hour window starting at start_hour). If not specified all days of the week will be used.
  
   * - start_hour
     - Integer
     - An integer between 0 and 23 denoting the start hour for the restart window in the 24-hour day. Continuous pipeline restart is triggered only within a five-hour window starting at this hour.
  
   * - time_zone_id
     - String
     - Time zone id of restart window. See https://docs.databricks.com/sql/language-manual/sql-ref-syntax-aux-conf-mgmt-set-timezone.html for details. If not specified, UTC will be used.
  
  
### resources.pipelines.trigger
Which pipeline trigger to use. Deprecated: Use `continuous` instead.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - cron
     - Map
     - 
  
   * - manual
     - Map
     - 
  
  
### resources.pipelines.trigger.cron

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - quartz_cron_schedule
     - String
     - 
  
   * - timezone_id
     - String
     - 
  
  
### resources.quality_monitors
The quality monitor definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.quality_monitors-entry-name>
     - Map
     - Item of the `resources.quality_monitors` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - assets_dir
     - String
     - The directory to store monitoring assets (e.g. dashboard, metric tables).
  
   * - baseline_table_name
     - String
     - Name of the baseline table from which drift metrics are computed from. Columns in the monitored table should also be present in the baseline table. 
  
   * - custom_metrics
     - Sequence
     - Custom metrics to compute on the monitored table. These can be aggregate metrics, derived metrics (from already computed aggregate metrics), or drift metrics (comparing metrics across time windows). 
  
   * - data_classification_config
     - Map
     - The data classification config for the monitor.
  
   * - inference_log
     - Map
     - Configuration for monitoring inference logs.
  
   * - notifications
     - Map
     - The notification settings for the monitor.
  
   * - output_schema_name
     - String
     - Schema where output metric tables are created.
  
   * - schedule
     - Map
     - The schedule for automatically updating and refreshing metric tables.
  
   * - skip_builtin_dashboard
     - Boolean
     - Whether to skip creating a default dashboard summarizing data quality metrics.
  
   * - slicing_exprs
     - Sequence
     - List of column expressions to slice data with for targeted analysis. The data is grouped by each expression independently, resulting in a separate slice for each predicate and its complements. For high-cardinality columns, only the top 100 unique values by frequency will generate slices. 
  
   * - snapshot
     - Map
     - Configuration for monitoring snapshot tables.
  
   * - table_name
     - String
     - 
  
   * - time_series
     - Map
     - Configuration for monitoring time series tables.
  
   * - warehouse_id
     - String
     - Optional argument to specify the warehouse for dashboard creation. If not specified, the first running warehouse will be used. 
  
  
### resources.quality_monitors.data_classification_config
The data classification config for the monitor.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - Whether data classification is enabled.
  
  
### resources.quality_monitors.inference_log
Configuration for monitoring inference logs.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - granularities
     - Sequence
     - Granularities for aggregating data into time windows based on their timestamp. Currently the following static granularities are supported: {``"5 minutes"``, ``"30 minutes"``, ``"1 hour"``, ``"1 day"``, ``"<n> week(s)"``, ``"1 month"``, ``"1 year"``}. 
  
   * - label_col
     - String
     - Optional column that contains the ground truth for the prediction.
  
   * - model_id_col
     - String
     - Column that contains the id of the model generating the predictions. Metrics will be computed per model id by default, and also across all model ids. 
  
   * - prediction_col
     - String
     - Column that contains the output/prediction from the model.
  
   * - prediction_proba_col
     - String
     - Optional column that contains the prediction probabilities for each class in a classification problem type. The values in this column should be a map, mapping each class label to the prediction probability for a given sample. The map should be of PySpark MapType(). 
  
   * - problem_type
     - String
     - Problem type the model aims to solve. Determines the type of model-quality metrics that will be computed.
  
   * - timestamp_col
     - String
     - Column that contains the timestamps of requests. The column must be one of the following: - A ``TimestampType`` column - A column whose values can be converted to timestamps through the pyspark   ``to_timestamp`` [function](https://spark.apache.org/docs/latest/api/python/reference/pyspark.sql/api/pyspark.sql.functions.to_timestamp.html). 
  
  
### resources.quality_monitors.notifications
The notification settings for the monitor.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - on_failure
     - Map
     - Who to send notifications to on monitor failure.
  
   * - on_new_classification_tag_detected
     - Map
     - Who to send notifications to when new data classification tags are detected.
  
  
### resources.quality_monitors.notifications.on_failure
Who to send notifications to on monitor failure.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - email_addresses
     - Sequence
     - The list of email addresses to send the notification to. A maximum of 5 email addresses is supported.
  
  
### resources.quality_monitors.notifications.on_new_classification_tag_detected
Who to send notifications to when new data classification tags are detected.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - email_addresses
     - Sequence
     - The list of email addresses to send the notification to. A maximum of 5 email addresses is supported.
  
  
### resources.quality_monitors.schedule
The schedule for automatically updating and refreshing metric tables.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - pause_status
     - String
     - Read only field that indicates whether a schedule is paused or not.
  
   * - quartz_cron_expression
     - String
     - The expression that determines when to run the monitor. See [examples](https://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html). 
  
   * - timezone_id
     - String
     - The timezone id (e.g., ``"PST"``) in which to evaluate the quartz expression. 
  
  
### resources.quality_monitors.time_series
Configuration for monitoring time series tables.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - granularities
     - Sequence
     - Granularities for aggregating data into time windows based on their timestamp. Currently the following static granularities are supported: {``"5 minutes"``, ``"30 minutes"``, ``"1 hour"``, ``"1 day"``, ``"<n> week(s)"``, ``"1 month"``, ``"1 year"``}. 
  
   * - timestamp_col
     - String
     - Column that contains the timestamps of requests. The column must be one of the following: - A ``TimestampType`` column - A column whose values can be converted to timestamps through the pyspark   ``to_timestamp`` [function](https://spark.apache.org/docs/latest/api/python/reference/pyspark.sql/api/pyspark.sql.functions.to_timestamp.html). 
  
  
### resources.registered_models
The registered model definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.registered_models-entry-name>
     - Map
     - Item of the `resources.registered_models` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog where the schema and the registered model reside
  
   * - comment
     - String
     - The comment attached to the registered model
  
   * - grants
     - Sequence
     - 
  
   * - name
     - String
     - The name of the registered model
  
   * - schema_name
     - String
     - The name of the schema where the registered model resides
  
   * - storage_location
     - String
     - The storage location on the cloud under which model version data files are stored
  
  
### resources.schemas
The schema definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.schemas-entry-name>
     - Map
     - Item of the `resources.schemas` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - Name of parent catalog.
  
   * - comment
     - String
     - User-provided free-form text description.
  
   * - grants
     - Sequence
     - 
  
   * - name
     - String
     - Name of schema, relative to parent catalog.
  
   * - properties
     - Map
     - 
  
   * - storage_root
     - String
     - Storage root URL for managed tables within schema.
  
  
### resources.volumes

  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <resources.volumes-entry-name>
     - Map
     - Item of the `resources.volumes` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog where the schema and the volume are
  
   * - comment
     - String
     - The comment attached to the volume
  
   * - grants
     - Sequence
     - 
  
   * - name
     - String
     - The name of the volume
  
   * - schema_name
     - String
     - The name of the schema where the volume is
  
   * - storage_location
     - String
     - The storage location on the cloud
  
   * - volume_type
     - String
     - 
  
  
## run_as
The identity to use to run the bundle.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - service_principal_name
     - String
     - Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
  
   * - user_name
     - String
     - The email of an active workspace user. Non-admin users can only set this field to their own email.
  
  
## sync
The files and file paths to include or exclude in the bundle. See [_](/dev-tools/bundles/)
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - exclude
     - Sequence
     - A list of files or folders to exclude from the bundle.
  
   * - include
     - Sequence
     - A list of files or folders to include in the bundle.
  
   * - paths
     - Sequence
     - The local folder paths, which can be outside the bundle root, to synchronize to the workspace when the bundle is deployed.
  
  
## targets
Defines deployment targets for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets-entry-name>
     - Map
     - Item of the `targets` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - artifacts
     - Map
     - The artifacts to include in the target deployment. See [_](#artifact)
  
   * - bundle
     - Map
     - The name of the bundle when deploying to this target.
  
   * - cluster_id
     - String
     - The ID of the cluster to use for this target.
  
   * - compute_id
     - String
     - Deprecated. The ID of the compute to use for this target.
  
   * - default
     - Boolean
     - Whether this target is the default target.
  
   * - git
     - Map
     - The Git version control settings for the target. See [_](#git).
  
   * - mode
     - String
     - The deployment mode for the target. Valid values are `development` or `production`. See [_](/dev-tools/bundles/deployment-modes.md).
  
   * - permissions
     - Sequence
     - The permissions for deploying and running the bundle in the target. See [_](#permission).
  
   * - presets
     - Map
     - The deployment presets for the target. See [_](#preset).
  
   * - resources
     - Map
     - The resource definitions for the target. See [_](#resources).
  
   * - run_as
     - Map
     - The identity to use to run the bundle. See [_](#job_run_as) and [_](/dev-tools/bundles/run_as.md).
  
   * - sync
     - Map
     - The local paths to sync to the target workspace when a bundle is run or deployed. See [_](#sync).
  
   * - variables
     - Map
     - The custom variable definitions for the target. See [_](/dev-tools/bundles/settings.md#variables) and [_](/dev-tools/bundles/variables.md).
  
   * - workspace
     - Map
     - The Databricks workspace for the target. [_](#workspace)
  
  
### targets.artifacts
The artifacts to include in the target deployment.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.artifacts-entry-name>
     - Map
     - Item of the `targets.artifacts` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - build
     - String
     - An optional set of non-default build commands that you want to run locally before deployment.  For Python wheel builds, the Databricks CLI assumes that it can find a local install of the Python wheel package to run builds, and it runs the command python setup.py bdist_wheel by default during each bundle deployment.  To specify multiple build commands, separate each command with double-ampersand (&&) characters.
  
   * - executable
     - String
     - The executable type.
  
   * - files
     - Sequence
     - The source files for the artifact, defined as an [_](#artifact_file).
  
   * - path
     - String
     - The location where the built artifact will be saved.
  
   * - type
     - String
     - The type of the artifact. Valid values are `wheel` or `jar`
  
  
### targets.bundle
The name of the bundle when deploying to this target.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - cluster_id
     - String
     - The ID of a cluster to use to run the bundle. See [_](/dev-tools/bundles/settings.md#cluster_id).
  
   * - compute_id
     - String
     - 
  
   * - databricks_cli_version
     - String
     - The Databricks CLI version to use for the bundle. See [_](/dev-tools/bundles/settings.md#databricks_cli_version).
  
   * - deployment
     - Map
     - The definition of the bundle deployment. For supported attributes, see [_](#deployment) and [_](/dev-tools/bundles/deployment-modes.md).
  
   * - git
     - Map
     - The Git version control details that are associated with your bundle. For supported attributes, see [_](#git) and [_](/dev-tools/bundles/settings.md#git).
  
   * - name
     - String
     - The name of the bundle.
  
   * - uuid
     - String
     - 
  
  
### targets.bundle.deployment
The definition of the bundle deployment
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - fail_on_active_runs
     - Boolean
     - Whether to fail on active runs. If this is set to true a deployment that is running can be interrupted.
  
   * - lock
     - Map
     - The deployment lock attributes. See [_](#lock).
  
  
### targets.bundle.deployment.lock
The deployment lock attributes.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - Whether this lock is enabled.
  
   * - force
     - Boolean
     - Whether to force this lock if it is enabled.
  
  
### targets.bundle.git
The Git version control details that are associated with your bundle.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - branch
     - String
     - The Git branch name. See [_](/dev-tools/bundles/settings.md#git).
  
   * - origin_url
     - String
     - The origin URL of the repository. See [_](/dev-tools/bundles/settings.md#git).
  
  
### targets.git
The Git version control settings for the target.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - branch
     - String
     - The Git branch name. See [_](/dev-tools/bundles/settings.md#git).
  
   * - origin_url
     - String
     - The origin URL of the repository. See [_](/dev-tools/bundles/settings.md#git).
  
  
### targets.presets
The deployment presets for the target.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - jobs_max_concurrent_runs
     - Integer
     - The maximum concurrent runs for a job.
  
   * - name_prefix
     - String
     - The prefix for job runs of the bundle.
  
   * - pipelines_development
     - Boolean
     - Whether pipeline deployments should be locked in development mode.
  
   * - source_linked_deployment
     - Boolean
     - Whether to link the deployment to the bundle source.
  
   * - tags
     - Map
     - The tags for the bundle deployment.
  
   * - trigger_pause_status
     - String
     - A pause status to apply to all job triggers and schedules. Valid values are PAUSED or UNPAUSED.
  
  
### targets.resources
The resource definitions for the target.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - clusters
     - Map
     - The cluster definitions for the bundle. See [_](/dev-tools/bundles/resources.md#cluster)
  
   * - dashboards
     - Map
     - The dashboard definitions for the bundle. See [_](/dev-tools/bundles/resources.md#dashboard)
  
   * - experiments
     - Map
     - The experiment definitions for the bundle. See [_](/dev-tools/bundles/resources.md#experiment)
  
   * - jobs
     - Map
     - The job definitions for the bundle. See [_](/dev-tools/bundles/resources.md#job)
  
   * - model_serving_endpoints
     - Map
     - The model serving endpoint definitions for the bundle. See [_](/dev-tools/bundles/resources.md#model_serving_endpoint)
  
   * - models
     - Map
     - The model definitions for the bundle. See [_](/dev-tools/bundles/resources.md#model)
  
   * - pipelines
     - Map
     - The pipeline definitions for the bundle. See [_](/dev-tools/bundles/resources.md#pipeline)
  
   * - quality_monitors
     - Map
     - The quality monitor definitions for the bundle. See [_](/dev-tools/bundles/resources.md#quality_monitor)
  
   * - registered_models
     - Map
     - The registered model definitions for the bundle. See [_](/dev-tools/bundles/resources.md#registered_model)
  
   * - schemas
     - Map
     - The schema definitions for the bundle. See [_](/dev-tools/bundles/resources.md#schema)
  
   * - volumes
     - Map
     - 
  
  
### targets.resources.clusters
The cluster definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.clusters-entry-name>
     - Map
     - Item of the `targets.resources.clusters` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - apply_policy_default_values
     - Boolean
     - When set to true, fixed and default values from the policy will be used for fields that are omitted. When set to false, only fixed values from the policy will be applied.
  
   * - autoscale
     - Map
     - Parameters needed in order to automatically scale clusters up and down based on load. Note: autoscaling works best with DB runtime versions 3.0 or later.
  
   * - autotermination_minutes
     - Integer
     - Automatically terminates the cluster after it is inactive for this time in minutes. If not set, this cluster will not be automatically terminated. If specified, the threshold must be between 10 and 10000 minutes. Users can also set this value to 0 to explicitly disable automatic termination.
  
   * - aws_attributes
     - Map
     - Attributes related to clusters running on Amazon Web Services. If not specified at cluster creation, a set of default values will be used.
  
   * - azure_attributes
     - Map
     - Attributes related to clusters running on Microsoft Azure. If not specified at cluster creation, a set of default values will be used.
  
   * - cluster_log_conf
     - Map
     - The configuration for delivering spark logs to a long-term storage destination. Two kinds of destinations (dbfs and s3) are supported. Only one destination can be specified for one cluster. If the conf is given, the logs will be delivered to the destination every `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while the destination of executor logs is `$destination/$clusterId/executor`.
  
   * - cluster_name
     - String
     - Cluster name requested by the user. This doesn't have to be unique. If not specified at creation, the cluster name will be an empty string. 
  
   * - custom_tags
     - Map
     - Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS instances and EBS volumes) with these tags in addition to `default_tags`. Notes:  - Currently, Databricks allows at most 45 custom tags  - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
  
   * - data_security_mode
     - String
     - 
  
   * - docker_image
     - Map
     - 
  
   * - driver_instance_pool_id
     - String
     - The optional ID of the instance pool for the driver of the cluster belongs. The pool cluster uses the instance pool with id (instance_pool_id) if the driver pool is not assigned.
  
   * - driver_node_type_id
     - String
     - The node type of the Spark driver. Note that this field is optional; if unset, the driver node type will be set as the same value as `node_type_id` defined above. 
  
   * - enable_elastic_disk
     - Boolean
     - Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space. This feature requires specific AWS permissions to function correctly - refer to the User Guide for more details.
  
   * - enable_local_disk_encryption
     - Boolean
     - Whether to enable LUKS on cluster VMs' local disks
  
   * - gcp_attributes
     - Map
     - Attributes related to clusters running on Google Cloud Platform. If not specified at cluster creation, a set of default values will be used.
  
   * - init_scripts
     - Sequence
     - The configuration for storing init scripts. Any number of destinations can be specified. The scripts are executed sequentially in the order provided. If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
  
   * - instance_pool_id
     - String
     - The optional ID of the instance pool to which the cluster belongs.
  
   * - node_type_id
     - String
     - This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster. For example, the Spark nodes can be provisioned and optimized for memory or compute intensive workloads. A list of available node types can be retrieved by using the :method:clusters/listNodeTypes API call. 
  
   * - num_workers
     - Integer
     - Number of worker nodes that this cluster should have. A cluster has one Spark Driver and `num_workers` Executors for a total of `num_workers` + 1 Spark nodes.  Note: When reading the properties of a cluster, this field reflects the desired number of workers rather than the actual current number of workers. For instance, if a cluster is resized from 5 to 10 workers, this field will immediately be updated to reflect the target size of 10 workers, whereas the workers listed in `spark_info` will gradually increase from 5 to 10 as the new nodes are provisioned.
  
   * - permissions
     - Sequence
     - 
  
   * - policy_id
     - String
     - The ID of the cluster policy used to create the cluster if applicable.
  
   * - runtime_engine
     - String
     - 
  
   * - single_user_name
     - String
     - Single user name if data_security_mode is `SINGLE_USER`
  
   * - spark_conf
     - Map
     - An object containing a set of optional, user-specified Spark configuration key-value pairs. Users can also pass in a string of extra JVM options to the driver and the executors via `spark.driver.extraJavaOptions` and `spark.executor.extraJavaOptions` respectively. 
  
   * - spark_env_vars
     - Map
     - An object containing a set of optional, user-specified environment variable key-value pairs. Please note that key-value pair of the form (X,Y) will be exported as is (i.e., `export X='Y'`) while launching the driver and workers.  In order to specify an additional set of `SPARK_DAEMON_JAVA_OPTS`, we recommend appending them to `$SPARK_DAEMON_JAVA_OPTS` as shown in the example below. This ensures that all default databricks managed environmental variables are included as well.  Example Spark environment variables: `{"SPARK_WORKER_MEMORY": "28000m", "SPARK_LOCAL_DIRS": "/local_disk0"}` or `{"SPARK_DAEMON_JAVA_OPTS": "$SPARK_DAEMON_JAVA_OPTS -Dspark.shuffle.service.enabled=true"}`
  
   * - spark_version
     - String
     - The Spark version of the cluster, e.g. `3.3.x-scala2.11`. A list of available Spark versions can be retrieved by using the :method:clusters/sparkVersions API call. 
  
   * - ssh_public_keys
     - Sequence
     - SSH public key contents that will be added to each Spark node in this cluster. The corresponding private keys can be used to login with the user name `ubuntu` on port `2200`. Up to 10 keys can be specified.
  
   * - workload_type
     - Map
     - 
  
  
### targets.resources.clusters.autoscale
Parameters needed in order to automatically scale clusters up and down based on load.
Note: autoscaling works best with DB runtime versions 3.0 or later.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - max_workers
     - Integer
     - The maximum number of workers to which the cluster can scale up when overloaded. Note that `max_workers` must be strictly greater than `min_workers`.
  
   * - min_workers
     - Integer
     - The minimum number of workers to which the cluster can scale down when underutilized. It is also the initial number of workers the cluster will have after creation.
  
  
### targets.resources.clusters.aws_attributes
Attributes related to clusters running on Amazon Web Services.
If not specified at cluster creation, a set of default values will be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - availability
     - String
     - 
  
   * - ebs_volume_count
     - Integer
     - The number of volumes launched for each instance. Users can choose up to 10 volumes. This feature is only enabled for supported node types. Legacy node types cannot specify custom EBS volumes. For node types with no instance store, at least one EBS volume needs to be specified; otherwise, cluster creation will fail.  These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc. Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.  If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for scratch storage because heterogenously sized scratch devices can lead to inefficient disk utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance store volumes.  Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir` will be overridden.
  
   * - ebs_volume_iops
     - Integer
     - If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
  
   * - ebs_volume_size
     - Integer
     - The size of each EBS volume (in GiB) launched for each instance. For general purpose SSD, this value must be within the range 100 - 4096. For throughput optimized HDD, this value must be within the range 500 - 4096.
  
   * - ebs_volume_throughput
     - Integer
     - If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
  
   * - ebs_volume_type
     - String
     - 
  
   * - first_on_demand
     - Integer
     - The first `first_on_demand` nodes of the cluster will be placed on on-demand instances. If this value is greater than 0, the cluster driver node in particular will be placed on an on-demand instance. If this value is greater than or equal to the current cluster size, all nodes will be placed on on-demand instances. If this value is less than the current cluster size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will be placed on `availability` instances. Note that this value does not affect cluster size and cannot currently be mutated over the lifetime of a cluster.
  
   * - instance_profile_arn
     - String
     - Nodes for this cluster will only be placed on AWS instances with this instance profile. If ommitted, nodes will be placed on instances without an IAM instance profile. The instance profile must have previously been added to the Databricks environment by an account administrator.  This feature may only be available to certain customer plans.  If this field is ommitted, we will pull in the default from the conf if it exists.
  
   * - spot_bid_price_percent
     - Integer
     - The bid price for AWS spot instances, as a percentage of the corresponding instance type's on-demand price. For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot instance, then the bid price is half of the price of on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100. When spot instances are requested for this cluster, only spot instances whose bid price percentage matches this field will be considered. Note that, for safety, we enforce this field to be no more than 10000.  The default value and documentation here should be kept consistent with CommonConf.defaultSpotBidPricePercent and CommonConf.maxSpotBidPricePercent.
  
   * - zone_id
     - String
     - Identifier for the availability zone/datacenter in which the cluster resides. This string will be of a form like "us-west-2a". The provided availability zone must be in the same region as the Databricks deployment. For example, "us-west-2a" is not a valid zone id if the Databricks deployment resides in the "us-east-1" region. This is an optional field at cluster creation, and if not specified, a default zone will be used. If the zone specified is "auto", will try to place cluster in a zone with high availability, and will retry placement in a different AZ if there is not enough capacity. The list of available zones as well as the default value can be found by using the `List Zones` method.
  
  
### targets.resources.clusters.azure_attributes
Attributes related to clusters running on Microsoft Azure.
If not specified at cluster creation, a set of default values will be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - availability
     - String
     - 
  
   * - first_on_demand
     - Integer
     - The first `first_on_demand` nodes of the cluster will be placed on on-demand instances. This value should be greater than 0, to make sure the cluster driver node is placed on an on-demand instance. If this value is greater than or equal to the current cluster size, all nodes will be placed on on-demand instances. If this value is less than the current cluster size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will be placed on `availability` instances. Note that this value does not affect cluster size and cannot currently be mutated over the lifetime of a cluster.
  
   * - log_analytics_info
     - Map
     - Defines values necessary to configure and run Azure Log Analytics agent
  
   * - spot_bid_max_price
     - Any
     - The max bid price to be used for Azure spot instances. The Max price for the bid cannot be higher than the on-demand price of the instance. If not specified, the default value is -1, which specifies that the instance cannot be evicted on the basis of price, and only on the basis of availability. Further, the value should > 0 or -1.
  
  
### targets.resources.clusters.azure_attributes.log_analytics_info
Defines values necessary to configure and run Azure Log Analytics agent
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - log_analytics_primary_key
     - String
     - <needs content added>
  
   * - log_analytics_workspace_id
     - String
     - <needs content added>
  
  
### targets.resources.clusters.cluster_log_conf
The configuration for delivering spark logs to a long-term storage destination.
Two kinds of destinations (dbfs and s3) are supported. Only one destination can be specified
for one cluster. If the conf is given, the logs will be delivered to the destination every
`5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
the destination of executor logs is `$destination/$clusterId/executor`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - dbfs
     - Map
     - destination needs to be provided. e.g. `{ "dbfs" : { "destination" : "dbfs:/home/cluster_log" } }`
  
   * - s3
     - Map
     - destination and either the region or endpoint need to be provided. e.g. `{ "s3": { "destination" : "s3://cluster_log_bucket/prefix", "region" : "us-west-2" } }` Cluster iam role is used to access s3, please make sure the cluster iam role in `instance_profile_arn` has permission to write data to the s3 destination.
  
  
### targets.resources.clusters.cluster_log_conf.dbfs
destination needs to be provided. e.g.
`{ "dbfs" : { "destination" : "dbfs:/home/cluster_log" } }`
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - destination
     - String
     - dbfs destination, e.g. `dbfs:/my/path`
  
  
### targets.resources.clusters.cluster_log_conf.s3
destination and either the region or endpoint need to be provided. e.g.
`{ "s3": { "destination" : "s3://cluster_log_bucket/prefix", "region" : "us-west-2" } }`
Cluster iam role is used to access s3, please make sure the cluster iam role in
`instance_profile_arn` has permission to write data to the s3 destination.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - canned_acl
     - String
     - (Optional) Set canned access control list for the logs, e.g. `bucket-owner-full-control`. If `canned_cal` is set, please make sure the cluster iam role has `s3:PutObjectAcl` permission on the destination bucket and prefix. The full list of possible canned acl can be found at http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl. Please also note that by default only the object owner gets full controls. If you are using cross account role for writing data, you may want to set `bucket-owner-full-control` to make bucket owner able to read the logs.
  
   * - destination
     - String
     - S3 destination, e.g. `s3://my-bucket/some-prefix` Note that logs will be delivered using cluster iam role, please make sure you set cluster iam role and the role has write access to the destination. Please also note that you cannot use AWS keys to deliver logs.
  
   * - enable_encryption
     - Boolean
     - (Optional) Flag to enable server side encryption, `false` by default.
  
   * - encryption_type
     - String
     - (Optional) The encryption type, it could be `sse-s3` or `sse-kms`. It will be used only when encryption is enabled and the default type is `sse-s3`.
  
   * - endpoint
     - String
     - S3 endpoint, e.g. `https://s3-us-west-2.amazonaws.com`. Either region or endpoint needs to be set. If both are set, endpoint will be used.
  
   * - kms_key
     - String
     - (Optional) Kms key which will be used if encryption is enabled and encryption type is set to `sse-kms`.
  
   * - region
     - String
     - S3 region, e.g. `us-west-2`. Either region or endpoint needs to be set. If both are set, endpoint will be used.
  
  
### targets.resources.clusters.docker_image

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - basic_auth
     - Map
     - 
  
   * - url
     - String
     - URL of the docker image.
  
  
### targets.resources.clusters.docker_image.basic_auth

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - password
     - String
     - Password of the user
  
   * - username
     - String
     - Name of the user
  
  
### targets.resources.clusters.gcp_attributes
Attributes related to clusters running on Google Cloud Platform.
If not specified at cluster creation, a set of default values will be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - availability
     - String
     - 
  
   * - boot_disk_size
     - Integer
     - boot disk size in GB
  
   * - google_service_account
     - String
     - If provided, the cluster will impersonate the google service account when accessing gcloud services (like GCS). The google service account must have previously been added to the Databricks environment by an account administrator.
  
   * - local_ssd_count
     - Integer
     - If provided, each node (workers and driver) in the cluster will have this number of local SSDs attached. Each local SSD is 375GB in size. Refer to [GCP documentation](https://cloud.google.com/compute/docs/disks/local-ssd#choose_number_local_ssds) for the supported number of local SSDs for each instance type.
  
   * - use_preemptible_executors
     - Boolean
     - This field determines whether the spark executors will be scheduled to run on preemptible VMs (when set to true) versus standard compute engine VMs (when set to false; default). Note: Soon to be deprecated, use the availability field instead.
  
   * - zone_id
     - String
     - Identifier for the availability zone in which the cluster resides. This can be one of the following: - "HA" => High availability, spread nodes across availability zones for a Databricks deployment region [default] - "AUTO" => Databricks picks an availability zone to schedule the cluster on. - A GCP availability zone => Pick One of the available zones for (machine type + region) from https://cloud.google.com/compute/docs/regions-zones.
  
  
### targets.resources.clusters.workload_type

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - clients
     - Map
     -  defined what type of clients can use the cluster. E.g. Notebooks, Jobs
  
  
### targets.resources.clusters.workload_type.clients
 defined what type of clients can use the cluster. E.g. Notebooks, Jobs
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - jobs
     - Boolean
     - With jobs set, the cluster can be used for jobs
  
   * - notebooks
     - Boolean
     - With notebooks set, this cluster can be used for notebooks
  
  
### targets.resources.dashboards
The dashboard definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.dashboards-entry-name>
     - Map
     - Item of the `targets.resources.dashboards` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - create_time
     - String
     - The timestamp of when the dashboard was created.
  
   * - dashboard_id
     - String
     - UUID identifying the dashboard.
  
   * - display_name
     - String
     - The display name of the dashboard.
  
   * - embed_credentials
     - Boolean
     - 
  
   * - etag
     - String
     - The etag for the dashboard. Can be optionally provided on updates to ensure that the dashboard has not been modified since the last read. This field is excluded in List Dashboards responses.
  
   * - file_path
     - String
     - 
  
   * - lifecycle_state
     - String
     - The state of the dashboard resource. Used for tracking trashed status.
  
   * - parent_path
     - String
     - The workspace path of the folder containing the dashboard. Includes leading slash and no trailing slash. This field is excluded in List Dashboards responses.
  
   * - path
     - String
     - The workspace path of the dashboard asset, including the file name. Exported dashboards always have the file extension `.lvdash.json`. This field is excluded in List Dashboards responses.
  
   * - permissions
     - Sequence
     - 
  
   * - serialized_dashboard
     - Any
     - The contents of the dashboard in serialized string form. This field is excluded in List Dashboards responses. Use the [get dashboard API](https://docs.databricks.com/api/workspace/lakeview/get) to retrieve an example response, which includes the `serialized_dashboard` field. This field provides the structure of the JSON string that represents the dashboard's layout and components.
  
   * - update_time
     - String
     - The timestamp of when the dashboard was last updated by the user. This field is excluded in List Dashboards responses.
  
   * - warehouse_id
     - String
     - The warehouse ID used to run the dashboard.
  
  
### targets.resources.experiments
The experiment definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.experiments-entry-name>
     - Map
     - Item of the `targets.resources.experiments` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - artifact_location
     - String
     - Location where artifacts for the experiment are stored.
  
   * - creation_time
     - Integer
     - Creation time
  
   * - experiment_id
     - String
     - Unique identifier for the experiment.
  
   * - last_update_time
     - Integer
     - Last update time
  
   * - lifecycle_stage
     - String
     - Current life cycle stage of the experiment: "active" or "deleted". Deleted experiments are not returned by APIs.
  
   * - name
     - String
     - Human readable name that identifies the experiment.
  
   * - permissions
     - Sequence
     - 
  
   * - tags
     - Sequence
     - Tags: Additional metadata key-value pairs.
  
  
### targets.resources.jobs
The job definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.jobs-entry-name>
     - Map
     - Item of the `targets.resources.jobs` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - budget_policy_id
     - String
     - The id of the user specified budget policy to use for this job. If not specified, a default budget policy may be applied when creating or modifying the job. See `effective_budget_policy_id` for the budget policy used by this workload.
  
   * - continuous
     - Map
     - An optional continuous property for this job. The continuous property will ensure that there is always one run executing. Only one of `schedule` and `continuous` can be used.
  
   * - description
     - String
     - An optional description for the job. The maximum length is 27700 characters in UTF-8 encoding.
  
   * - email_notifications
     - Map
     - An optional set of email addresses that is notified when runs of this job begin or complete as well as when this job is deleted.
  
   * - environments
     - Sequence
     - A list of task execution environment specifications that can be referenced by serverless tasks of this job. An environment is required to be present for serverless tasks. For serverless notebook tasks, the environment is accessible in the notebook environment panel. For other serverless tasks, the task environment is required to be specified using environment_key in the task settings.
  
   * - git_source
     - Map
     - An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.  If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.  Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
  
   * - health
     - Map
     - 
  
   * - job_clusters
     - Sequence
     - A list of job cluster specifications that can be shared and reused by tasks of this job. Libraries cannot be declared in a shared job cluster. You must declare dependent libraries in task settings.
  
   * - max_concurrent_runs
     - Integer
     - An optional maximum allowed number of concurrent runs of the job. Set this value if you want to be able to execute multiple runs of the same job concurrently. This is useful for example if you trigger your job on a frequent schedule and want to allow consecutive runs to overlap with each other, or if you want to trigger multiple runs which differ by their input parameters. This setting affects only new runs. For example, suppose the job’s concurrency is 4 and there are 4 concurrent active runs. Then setting the concurrency to 3 won’t kill any of the active runs. However, from then on, new runs are skipped unless there are fewer than 3 active runs. This value cannot exceed 1000. Setting this value to `0` causes all new runs to be skipped.
  
   * - name
     - String
     - An optional name for the job. The maximum length is 4096 bytes in UTF-8 encoding.
  
   * - notification_settings
     - Map
     - Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this job.
  
   * - parameters
     - Sequence
     - Job-level parameter definitions
  
   * - permissions
     - Sequence
     - 
  
   * - queue
     - Map
     - The queue settings of the job.
  
   * - run_as
     - Map
     - 
  
   * - schedule
     - Map
     - An optional periodic schedule for this job. The default behavior is that the job only runs when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
   * - tags
     - Map
     - A map of tags associated with the job. These are forwarded to the cluster as cluster tags for jobs clusters, and are subject to the same limitations as cluster tags. A maximum of 25 tags can be added to the job.
  
   * - tasks
     - Sequence
     - A list of task specifications to be executed by this job.
  
   * - timeout_seconds
     - Integer
     - An optional timeout applied to each run of this job. A value of `0` means no timeout.
  
   * - trigger
     - Map
     - A configuration to trigger a run when certain conditions are met. The default behavior is that the job runs only when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
   * - webhook_notifications
     - Map
     - A collection of system notification IDs to notify when runs of this job begin or complete.
  
  
### targets.resources.jobs.continuous
An optional continuous property for this job. The continuous property will ensure that there is always one run executing. Only one of `schedule` and `continuous` can be used.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - pause_status
     - String
     - Indicate whether the continuous execution of the job is paused or not. Defaults to UNPAUSED.
  
  
### targets.resources.jobs.email_notifications
An optional set of email addresses that is notified when runs of this job begin or complete as well as when this job is deleted.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - no_alert_for_skipped_runs
     - Boolean
     - If true, do not send email to recipients specified in `on_failure` if the run is skipped. This field is `deprecated`. Please use the `notification_settings.no_alert_for_skipped_runs` field.
  
   * - on_duration_warning_threshold_exceeded
     - Sequence
     - A list of email addresses to be notified when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. If no rule for the `RUN_DURATION_SECONDS` metric is specified in the `health` field for the job, notifications are not sent.
  
   * - on_failure
     - Sequence
     - A list of email addresses to be notified when a run unsuccessfully completes. A run is considered to have completed unsuccessfully if it ends with an `INTERNAL_ERROR` `life_cycle_state` or a `FAILED`, or `TIMED_OUT` result_state. If this is not specified on job creation, reset, or update the list is empty, and notifications are not sent.
  
   * - on_start
     - Sequence
     - A list of email addresses to be notified when a run begins. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
  
   * - on_streaming_backlog_exceeded
     - Sequence
     - A list of email addresses to notify when any streaming backlog thresholds are exceeded for any stream. Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`. Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
  
   * - on_success
     - Sequence
     - A list of email addresses to be notified when a run successfully completes. A run is considered to have completed successfully if it ends with a `TERMINATED` `life_cycle_state` and a `SUCCESS` result_state. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
  
  
### targets.resources.jobs.git_source
An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.

If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.

Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - git_branch
     - String
     - Name of the branch to be checked out and used by this job. This field cannot be specified in conjunction with git_tag or git_commit.
  
   * - git_commit
     - String
     - Commit to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_tag.
  
   * - git_provider
     - String
     - Unique identifier of the service used to host the Git repository. The value is case insensitive.
  
   * - git_tag
     - String
     - Name of the tag to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_commit.
  
   * - git_url
     - String
     - URL of the repository to be cloned by this job.
  
  
### targets.resources.jobs.health

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - rules
     - Sequence
     - 
  
  
### targets.resources.jobs.notification_settings
Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this job.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - no_alert_for_canceled_runs
     - Boolean
     - If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
  
   * - no_alert_for_skipped_runs
     - Boolean
     - If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
  
  
### targets.resources.jobs.queue
The queue settings of the job.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - If true, enable queueing for the job. This is a required field.
  
  
### targets.resources.jobs.run_as

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - service_principal_name
     - String
     - Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
  
   * - user_name
     - String
     - The email of an active workspace user. Non-admin users can only set this field to their own email.
  
  
### targets.resources.jobs.schedule
An optional periodic schedule for this job. The default behavior is that the job only runs when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - pause_status
     - String
     - Indicate whether this schedule is paused or not.
  
   * - quartz_cron_expression
     - String
     - A Cron expression using Quartz syntax that describes the schedule for a job. See [Cron Trigger](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html) for details. This field is required.
  
   * - timezone_id
     - String
     - A Java timezone ID. The schedule for a job is resolved with respect to this timezone. See [Java TimeZone](https://docs.oracle.com/javase/7/docs/api/java/util/TimeZone.html) for details. This field is required.
  
  
### targets.resources.jobs.trigger
A configuration to trigger a run when certain conditions are met. The default behavior is that the job runs only when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - file_arrival
     - Map
     - File arrival trigger settings.
  
   * - pause_status
     - String
     - Whether this trigger is paused or not.
  
   * - periodic
     - Map
     - Periodic trigger settings.
  
   * - table
     - Map
     - Old table trigger settings name. Deprecated in favor of `table_update`.
  
   * - table_update
     - Map
     - 
  
  
### targets.resources.jobs.trigger.file_arrival
File arrival trigger settings.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - min_time_between_triggers_seconds
     - Integer
     - If set, the trigger starts a run only after the specified amount of time passed since the last time the trigger fired. The minimum allowed value is 60 seconds
  
   * - url
     - String
     - URL to be monitored for file arrivals. The path must point to the root or a subpath of the external location.
  
   * - wait_after_last_change_seconds
     - Integer
     - If set, the trigger starts a run only after no file activity has occurred for the specified amount of time. This makes it possible to wait for a batch of incoming files to arrive before triggering a run. The minimum allowed value is 60 seconds.
  
  
### targets.resources.jobs.trigger.periodic
Periodic trigger settings.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - interval
     - Integer
     - The interval at which the trigger should run.
  
   * - unit
     - String
     - The unit of time for the interval.
  
  
### targets.resources.jobs.trigger.table
Old table trigger settings name. Deprecated in favor of `table_update`.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - condition
     - String
     - The table(s) condition based on which to trigger a job run.
  
   * - min_time_between_triggers_seconds
     - Integer
     - If set, the trigger starts a run only after the specified amount of time has passed since the last time the trigger fired. The minimum allowed value is 60 seconds.
  
   * - table_names
     - Sequence
     - A list of Delta tables to monitor for changes. The table name must be in the format `catalog_name.schema_name.table_name`.
  
   * - wait_after_last_change_seconds
     - Integer
     - If set, the trigger starts a run only after no table updates have occurred for the specified time and can be used to wait for a series of table updates before triggering a run. The minimum allowed value is 60 seconds.
  
  
### targets.resources.jobs.trigger.table_update

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - condition
     - String
     - The table(s) condition based on which to trigger a job run.
  
   * - min_time_between_triggers_seconds
     - Integer
     - If set, the trigger starts a run only after the specified amount of time has passed since the last time the trigger fired. The minimum allowed value is 60 seconds.
  
   * - table_names
     - Sequence
     - A list of Delta tables to monitor for changes. The table name must be in the format `catalog_name.schema_name.table_name`.
  
   * - wait_after_last_change_seconds
     - Integer
     - If set, the trigger starts a run only after no table updates have occurred for the specified time and can be used to wait for a series of table updates before triggering a run. The minimum allowed value is 60 seconds.
  
  
### targets.resources.jobs.webhook_notifications
A collection of system notification IDs to notify when runs of this job begin or complete.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - on_duration_warning_threshold_exceeded
     - Sequence
     - An optional list of system notification IDs to call when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. A maximum of 3 destinations can be specified for the `on_duration_warning_threshold_exceeded` property.
  
   * - on_failure
     - Sequence
     - An optional list of system notification IDs to call when the run fails. A maximum of 3 destinations can be specified for the `on_failure` property.
  
   * - on_start
     - Sequence
     - An optional list of system notification IDs to call when the run starts. A maximum of 3 destinations can be specified for the `on_start` property.
  
   * - on_streaming_backlog_exceeded
     - Sequence
     - An optional list of system notification IDs to call when any streaming backlog thresholds are exceeded for any stream. Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`. Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes. A maximum of 3 destinations can be specified for the `on_streaming_backlog_exceeded` property.
  
   * - on_success
     - Sequence
     - An optional list of system notification IDs to call when the run completes successfully. A maximum of 3 destinations can be specified for the `on_success` property.
  
  
### targets.resources.model_serving_endpoints
The model serving endpoint definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.model_serving_endpoints-entry-name>
     - Map
     - Item of the `targets.resources.model_serving_endpoints` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - ai_gateway
     - Map
     - The AI Gateway configuration for the serving endpoint. NOTE: only external model endpoints are supported as of now.
  
   * - config
     - Map
     - The core config of the serving endpoint.
  
   * - name
     - String
     - The name of the serving endpoint. This field is required and must be unique across a Databricks workspace. An endpoint name can consist of alphanumeric characters, dashes, and underscores. 
  
   * - permissions
     - Sequence
     - 
  
   * - rate_limits
     - Sequence
     - Rate limits to be applied to the serving endpoint. NOTE: this field is deprecated, please use AI Gateway to manage rate limits.
  
   * - route_optimized
     - Boolean
     - Enable route optimization for the serving endpoint.
  
   * - tags
     - Sequence
     - Tags to be attached to the serving endpoint and automatically propagated to billing logs.
  
  
### targets.resources.model_serving_endpoints.ai_gateway
The AI Gateway configuration for the serving endpoint. NOTE: only external model endpoints are supported as of now.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - guardrails
     - Map
     - Configuration for AI Guardrails to prevent unwanted data and unsafe data in requests and responses.
  
   * - inference_table_config
     - Map
     - Configuration for payload logging using inference tables. Use these tables to monitor and audit data being sent to and received from model APIs and to improve model quality.
  
   * - rate_limits
     - Sequence
     - Configuration for rate limits which can be set to limit endpoint traffic.
  
   * - usage_tracking_config
     - Map
     - Configuration to enable usage tracking using system tables. These tables allow you to monitor operational usage on endpoints and their associated costs.
  
  
### targets.resources.model_serving_endpoints.ai_gateway.guardrails
Configuration for AI Guardrails to prevent unwanted data and unsafe data in requests and responses.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - input
     - Map
     - Configuration for input guardrail filters.
  
   * - output
     - Map
     - Configuration for output guardrail filters.
  
  
### targets.resources.model_serving_endpoints.ai_gateway.guardrails.input
Configuration for input guardrail filters.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - invalid_keywords
     - Sequence
     - List of invalid keywords. AI guardrail uses keyword or string matching to decide if the keyword exists in the request or response content.
  
   * - pii
     - Map
     - Configuration for guardrail PII filter.
  
   * - safety
     - Boolean
     - Indicates whether the safety filter is enabled.
  
   * - valid_topics
     - Sequence
     - The list of allowed topics. Given a chat request, this guardrail flags the request if its topic is not in the allowed topics.
  
  
### targets.resources.model_serving_endpoints.ai_gateway.guardrails.input.pii
Configuration for guardrail PII filter.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - behavior
     - String
     - Behavior for PII filter. Currently only 'BLOCK' is supported. If 'BLOCK' is set for the input guardrail and the request contains PII, the request is not sent to the model server and 400 status code is returned; if 'BLOCK' is set for the output guardrail and the model response contains PII, the PII info in the response is redacted and 400 status code is returned.
  
  
### targets.resources.model_serving_endpoints.ai_gateway.guardrails.output
Configuration for output guardrail filters.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - invalid_keywords
     - Sequence
     - List of invalid keywords. AI guardrail uses keyword or string matching to decide if the keyword exists in the request or response content.
  
   * - pii
     - Map
     - Configuration for guardrail PII filter.
  
   * - safety
     - Boolean
     - Indicates whether the safety filter is enabled.
  
   * - valid_topics
     - Sequence
     - The list of allowed topics. Given a chat request, this guardrail flags the request if its topic is not in the allowed topics.
  
  
### targets.resources.model_serving_endpoints.ai_gateway.guardrails.output.pii
Configuration for guardrail PII filter.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - behavior
     - String
     - Behavior for PII filter. Currently only 'BLOCK' is supported. If 'BLOCK' is set for the input guardrail and the request contains PII, the request is not sent to the model server and 400 status code is returned; if 'BLOCK' is set for the output guardrail and the model response contains PII, the PII info in the response is redacted and 400 status code is returned.
  
  
### targets.resources.model_serving_endpoints.ai_gateway.inference_table_config
Configuration for payload logging using inference tables. Use these tables to monitor and audit data being sent to and received from model APIs and to improve model quality.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog in Unity Catalog. Required when enabling inference tables. NOTE: On update, you have to disable inference table first in order to change the catalog name.
  
   * - enabled
     - Boolean
     - Indicates whether the inference table is enabled.
  
   * - schema_name
     - String
     - The name of the schema in Unity Catalog. Required when enabling inference tables. NOTE: On update, you have to disable inference table first in order to change the schema name.
  
   * - table_name_prefix
     - String
     - The prefix of the table in Unity Catalog. NOTE: On update, you have to disable inference table first in order to change the prefix name.
  
  
### targets.resources.model_serving_endpoints.ai_gateway.usage_tracking_config
Configuration to enable usage tracking using system tables. These tables allow you to monitor operational usage on endpoints and their associated costs.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - Whether to enable usage tracking.
  
  
### targets.resources.model_serving_endpoints.config
The core config of the serving endpoint.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - auto_capture_config
     - Map
     - Configuration for Inference Tables which automatically logs requests and responses to Unity Catalog.
  
   * - served_entities
     - Sequence
     - A list of served entities for the endpoint to serve. A serving endpoint can have up to 15 served entities.
  
   * - served_models
     - Sequence
     - (Deprecated, use served_entities instead) A list of served models for the endpoint to serve. A serving endpoint can have up to 15 served models.
  
   * - traffic_config
     - Map
     - The traffic config defining how invocations to the serving endpoint should be routed.
  
  
### targets.resources.model_serving_endpoints.config.auto_capture_config
Configuration for Inference Tables which automatically logs requests and responses to Unity Catalog.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog in Unity Catalog. NOTE: On update, you cannot change the catalog name if the inference table is already enabled.
  
   * - enabled
     - Boolean
     - Indicates whether the inference table is enabled.
  
   * - schema_name
     - String
     - The name of the schema in Unity Catalog. NOTE: On update, you cannot change the schema name if the inference table is already enabled.
  
   * - table_name_prefix
     - String
     - The prefix of the table in Unity Catalog. NOTE: On update, you cannot change the prefix name if the inference table is already enabled.
  
  
### targets.resources.model_serving_endpoints.config.traffic_config
The traffic config defining how invocations to the serving endpoint should be routed.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - routes
     - Sequence
     - The list of routes that define traffic to each served entity.
  
  
### targets.resources.models
The model definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.models-entry-name>
     - Map
     - Item of the `targets.resources.models` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - creation_timestamp
     - Integer
     - Timestamp recorded when this `registered_model` was created.
  
   * - description
     - String
     - Description of this `registered_model`.
  
   * - last_updated_timestamp
     - Integer
     - Timestamp recorded when metadata for this `registered_model` was last updated.
  
   * - latest_versions
     - Sequence
     - Collection of latest model versions for each stage. Only contains models with current `READY` status.
  
   * - name
     - String
     - Unique name for the model.
  
   * - permissions
     - Sequence
     - 
  
   * - tags
     - Sequence
     - Tags: Additional metadata key-value pairs for this `registered_model`.
  
   * - user_id
     - String
     - User that created this `registered_model`
  
  
### targets.resources.pipelines
The pipeline definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.pipelines-entry-name>
     - Map
     - Item of the `targets.resources.pipelines` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - budget_policy_id
     - String
     - Budget policy of this pipeline.
  
   * - catalog
     - String
     - A catalog in Unity Catalog to publish data from this pipeline to. If `target` is specified, tables in this pipeline are published to a `target` schema inside `catalog` (for example, `catalog`.`target`.`table`). If `target` is not specified, no data is published to Unity Catalog.
  
   * - channel
     - String
     - DLT Release Channel that specifies which version to use.
  
   * - clusters
     - Sequence
     - Cluster settings for this pipeline deployment.
  
   * - configuration
     - Map
     - String-String configuration for this pipeline execution.
  
   * - continuous
     - Boolean
     - Whether the pipeline is continuous or triggered. This replaces `trigger`.
  
   * - deployment
     - Map
     - Deployment type of this pipeline.
  
   * - development
     - Boolean
     - Whether the pipeline is in Development mode. Defaults to false.
  
   * - edition
     - String
     - Pipeline product edition.
  
   * - filters
     - Map
     - Filters on which Pipeline packages to include in the deployed graph.
  
   * - gateway_definition
     - Map
     - The definition of a gateway pipeline to support change data capture.
  
   * - id
     - String
     - Unique identifier for this pipeline.
  
   * - ingestion_definition
     - Map
     - The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'target' or 'catalog' settings.
  
   * - libraries
     - Sequence
     - Libraries or code needed by this deployment.
  
   * - name
     - String
     - Friendly identifier for this pipeline.
  
   * - notifications
     - Sequence
     - List of notification settings for this pipeline.
  
   * - permissions
     - Sequence
     - 
  
   * - photon
     - Boolean
     - Whether Photon is enabled for this pipeline.
  
   * - restart_window
     - Map
     - Restart window of this pipeline.
  
   * - schema
     - String
     - The default schema (database) where tables are read from or published to. The presence of this field implies that the pipeline is in direct publishing mode.
  
   * - serverless
     - Boolean
     - Whether serverless compute is enabled for this pipeline.
  
   * - storage
     - String
     - DBFS root directory for storing checkpoints and tables.
  
   * - target
     - String
     - Target schema (database) to add tables in this pipeline to. If not specified, no data is published to the Hive metastore or Unity Catalog. To publish to Unity Catalog, also specify `catalog`.
  
   * - trigger
     - Map
     - Which pipeline trigger to use. Deprecated: Use `continuous` instead.
  
  
### targets.resources.pipelines.deployment
Deployment type of this pipeline.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - kind
     - String
     - The deployment method that manages the pipeline.
  
   * - metadata_file_path
     - String
     - The path to the file containing metadata about the deployment.
  
  
### targets.resources.pipelines.filters
Filters on which Pipeline packages to include in the deployed graph.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - exclude
     - Sequence
     - Paths to exclude.
  
   * - include
     - Sequence
     - Paths to include.
  
  
### targets.resources.pipelines.gateway_definition
The definition of a gateway pipeline to support change data capture.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - connection_id
     - String
     - [Deprecated, use connection_name instead] Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
  
   * - connection_name
     - String
     - Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
  
   * - gateway_storage_catalog
     - String
     - Required, Immutable. The name of the catalog for the gateway pipeline's storage location.
  
   * - gateway_storage_name
     - String
     - Optional. The Unity Catalog-compatible name for the gateway storage location. This is the destination to use for the data that is extracted by the gateway. Delta Live Tables system will automatically create the storage location under the catalog and schema. 
  
   * - gateway_storage_schema
     - String
     - Required, Immutable. The name of the schema for the gateway pipelines's storage location.
  
  
### targets.resources.pipelines.ingestion_definition
The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'target' or 'catalog' settings.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - connection_name
     - String
     - Immutable. The Unity Catalog connection that this ingestion pipeline uses to communicate with the source. This is used with connectors for applications like Salesforce, Workday, and so on.
  
   * - ingestion_gateway_id
     - String
     - Immutable. Identifier for the gateway that is used by this ingestion pipeline to communicate with the source database. This is used with connectors to databases like SQL Server.
  
   * - objects
     - Sequence
     - Required. Settings specifying tables to replicate and the destination for the replicated tables.
  
   * - table_configuration
     - Map
     - Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
  
  
### targets.resources.pipelines.ingestion_definition.table_configuration
Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - primary_keys
     - Sequence
     - The primary key of the table used to apply changes.
  
   * - salesforce_include_formula_fields
     - Boolean
     - If true, formula fields defined in the table are included in the ingestion. This setting is only valid for the Salesforce connector
  
   * - scd_type
     - String
     - The SCD type to use to ingest the table.
  
   * - sequence_by
     - Sequence
     - The column names specifying the logical order of events in the source data. Delta Live Tables uses this sequencing to handle change events that arrive out of order.
  
  
### targets.resources.pipelines.restart_window
Restart window of this pipeline.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - days_of_week
     - String
     - Days of week in which the restart is allowed to happen (within a five-hour window starting at start_hour). If not specified all days of the week will be used.
  
   * - start_hour
     - Integer
     - An integer between 0 and 23 denoting the start hour for the restart window in the 24-hour day. Continuous pipeline restart is triggered only within a five-hour window starting at this hour.
  
   * - time_zone_id
     - String
     - Time zone id of restart window. See https://docs.databricks.com/sql/language-manual/sql-ref-syntax-aux-conf-mgmt-set-timezone.html for details. If not specified, UTC will be used.
  
  
### targets.resources.pipelines.trigger
Which pipeline trigger to use. Deprecated: Use `continuous` instead.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - cron
     - Map
     - 
  
   * - manual
     - Map
     - 
  
  
### targets.resources.pipelines.trigger.cron

  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - quartz_cron_schedule
     - String
     - 
  
   * - timezone_id
     - String
     - 
  
  
### targets.resources.quality_monitors
The quality monitor definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.quality_monitors-entry-name>
     - Map
     - Item of the `targets.resources.quality_monitors` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - assets_dir
     - String
     - The directory to store monitoring assets (e.g. dashboard, metric tables).
  
   * - baseline_table_name
     - String
     - Name of the baseline table from which drift metrics are computed from. Columns in the monitored table should also be present in the baseline table. 
  
   * - custom_metrics
     - Sequence
     - Custom metrics to compute on the monitored table. These can be aggregate metrics, derived metrics (from already computed aggregate metrics), or drift metrics (comparing metrics across time windows). 
  
   * - data_classification_config
     - Map
     - The data classification config for the monitor.
  
   * - inference_log
     - Map
     - Configuration for monitoring inference logs.
  
   * - notifications
     - Map
     - The notification settings for the monitor.
  
   * - output_schema_name
     - String
     - Schema where output metric tables are created.
  
   * - schedule
     - Map
     - The schedule for automatically updating and refreshing metric tables.
  
   * - skip_builtin_dashboard
     - Boolean
     - Whether to skip creating a default dashboard summarizing data quality metrics.
  
   * - slicing_exprs
     - Sequence
     - List of column expressions to slice data with for targeted analysis. The data is grouped by each expression independently, resulting in a separate slice for each predicate and its complements. For high-cardinality columns, only the top 100 unique values by frequency will generate slices. 
  
   * - snapshot
     - Map
     - Configuration for monitoring snapshot tables.
  
   * - table_name
     - String
     - 
  
   * - time_series
     - Map
     - Configuration for monitoring time series tables.
  
   * - warehouse_id
     - String
     - Optional argument to specify the warehouse for dashboard creation. If not specified, the first running warehouse will be used. 
  
  
### targets.resources.quality_monitors.data_classification_config
The data classification config for the monitor.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - enabled
     - Boolean
     - Whether data classification is enabled.
  
  
### targets.resources.quality_monitors.inference_log
Configuration for monitoring inference logs.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - granularities
     - Sequence
     - Granularities for aggregating data into time windows based on their timestamp. Currently the following static granularities are supported: {``"5 minutes"``, ``"30 minutes"``, ``"1 hour"``, ``"1 day"``, ``"<n> week(s)"``, ``"1 month"``, ``"1 year"``}. 
  
   * - label_col
     - String
     - Optional column that contains the ground truth for the prediction.
  
   * - model_id_col
     - String
     - Column that contains the id of the model generating the predictions. Metrics will be computed per model id by default, and also across all model ids. 
  
   * - prediction_col
     - String
     - Column that contains the output/prediction from the model.
  
   * - prediction_proba_col
     - String
     - Optional column that contains the prediction probabilities for each class in a classification problem type. The values in this column should be a map, mapping each class label to the prediction probability for a given sample. The map should be of PySpark MapType(). 
  
   * - problem_type
     - String
     - Problem type the model aims to solve. Determines the type of model-quality metrics that will be computed.
  
   * - timestamp_col
     - String
     - Column that contains the timestamps of requests. The column must be one of the following: - A ``TimestampType`` column - A column whose values can be converted to timestamps through the pyspark   ``to_timestamp`` [function](https://spark.apache.org/docs/latest/api/python/reference/pyspark.sql/api/pyspark.sql.functions.to_timestamp.html). 
  
  
### targets.resources.quality_monitors.notifications
The notification settings for the monitor.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - on_failure
     - Map
     - Who to send notifications to on monitor failure.
  
   * - on_new_classification_tag_detected
     - Map
     - Who to send notifications to when new data classification tags are detected.
  
  
### targets.resources.quality_monitors.notifications.on_failure
Who to send notifications to on monitor failure.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - email_addresses
     - Sequence
     - The list of email addresses to send the notification to. A maximum of 5 email addresses is supported.
  
  
### targets.resources.quality_monitors.notifications.on_new_classification_tag_detected
Who to send notifications to when new data classification tags are detected.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - email_addresses
     - Sequence
     - The list of email addresses to send the notification to. A maximum of 5 email addresses is supported.
  
  
### targets.resources.quality_monitors.schedule
The schedule for automatically updating and refreshing metric tables.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - pause_status
     - String
     - Read only field that indicates whether a schedule is paused or not.
  
   * - quartz_cron_expression
     - String
     - The expression that determines when to run the monitor. See [examples](https://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html). 
  
   * - timezone_id
     - String
     - The timezone id (e.g., ``"PST"``) in which to evaluate the quartz expression. 
  
  
### targets.resources.quality_monitors.time_series
Configuration for monitoring time series tables.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - granularities
     - Sequence
     - Granularities for aggregating data into time windows based on their timestamp. Currently the following static granularities are supported: {``"5 minutes"``, ``"30 minutes"``, ``"1 hour"``, ``"1 day"``, ``"<n> week(s)"``, ``"1 month"``, ``"1 year"``}. 
  
   * - timestamp_col
     - String
     - Column that contains the timestamps of requests. The column must be one of the following: - A ``TimestampType`` column - A column whose values can be converted to timestamps through the pyspark   ``to_timestamp`` [function](https://spark.apache.org/docs/latest/api/python/reference/pyspark.sql/api/pyspark.sql.functions.to_timestamp.html). 
  
  
### targets.resources.registered_models
The registered model definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.registered_models-entry-name>
     - Map
     - Item of the `targets.resources.registered_models` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog where the schema and the registered model reside
  
   * - comment
     - String
     - The comment attached to the registered model
  
   * - grants
     - Sequence
     - 
  
   * - name
     - String
     - The name of the registered model
  
   * - schema_name
     - String
     - The name of the schema where the registered model resides
  
   * - storage_location
     - String
     - The storage location on the cloud under which model version data files are stored
  
  
### targets.resources.schemas
The schema definitions for the bundle.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.schemas-entry-name>
     - Map
     - Item of the `targets.resources.schemas` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - Name of parent catalog.
  
   * - comment
     - String
     - User-provided free-form text description.
  
   * - grants
     - Sequence
     - 
  
   * - name
     - String
     - Name of schema, relative to parent catalog.
  
   * - properties
     - Map
     - 
  
   * - storage_root
     - String
     - Storage root URL for managed tables within schema.
  
  
### targets.resources.volumes

  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.resources.volumes-entry-name>
     - Map
     - Item of the `targets.resources.volumes` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - catalog_name
     - String
     - The name of the catalog where the schema and the volume are
  
   * - comment
     - String
     - The comment attached to the volume
  
   * - grants
     - Sequence
     - 
  
   * - name
     - String
     - The name of the volume
  
   * - schema_name
     - String
     - The name of the schema where the volume is
  
   * - storage_location
     - String
     - The storage location on the cloud
  
   * - volume_type
     - String
     - 
  
  
### targets.run_as
The identity to use to run the bundle.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - service_principal_name
     - String
     - Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
  
   * - user_name
     - String
     - The email of an active workspace user. Non-admin users can only set this field to their own email.
  
  
### targets.sync
The local paths to sync to the target workspace when a bundle is run or deployed.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - exclude
     - Sequence
     - A list of files or folders to exclude from the bundle.
  
   * - include
     - Sequence
     - A list of files or folders to include in the bundle.
  
   * - paths
     - Sequence
     - The local folder paths, which can be outside the bundle root, to synchronize to the workspace when the bundle is deployed.
  
  
### targets.variables
The custom variable definitions for the target.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <targets.variables-entry-name>
     - Map
     - Item of the `targets.variables` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - default
     - Any
     - 
  
   * - description
     - String
     - The description of the variable.
  
   * - lookup
     - Map
     - The name of the alert, cluster_policy, cluster, dashboard, instance_pool, job, metastore, pipeline, query, service_principal, or warehouse object for which to retrieve an ID.
  
   * - type
     - String
     - The type of the variable.
  
  
### targets.variables.lookup
The name of the alert, cluster_policy, cluster, dashboard, instance_pool, job, metastore, pipeline, query, service_principal, or warehouse object for which to retrieve an ID.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - alert
     - String
     - 
  
   * - cluster
     - String
     - 
  
   * - cluster_policy
     - String
     - 
  
   * - dashboard
     - String
     - 
  
   * - instance_pool
     - String
     - 
  
   * - job
     - String
     - 
  
   * - metastore
     - String
     - 
  
   * - notification_destination
     - String
     - 
  
   * - pipeline
     - String
     - 
  
   * - query
     - String
     - 
  
   * - service_principal
     - String
     - 
  
   * - warehouse
     - String
     - 
  
  
### targets.workspace
The Databricks workspace for the target.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - artifact_path
     - String
     - The artifact path to use within the workspace for both deployments and workflow runs
  
   * - auth_type
     - String
     - The authentication type.
  
   * - azure_client_id
     - String
     - The Azure client ID
  
   * - azure_environment
     - String
     - The Azure environment
  
   * - azure_login_app_id
     - String
     - The Azure login app ID
  
   * - azure_tenant_id
     - String
     - The Azure tenant ID
  
   * - azure_use_msi
     - Boolean
     - Whether to use MSI for Azure
  
   * - azure_workspace_resource_id
     - String
     - The Azure workspace resource ID
  
   * - client_id
     - String
     - The client ID for the workspace
  
   * - file_path
     - String
     - The file path to use within the workspace for both deployments and workflow runs
  
   * - google_service_account
     - String
     - The Google service account name
  
   * - host
     - String
     - The Databricks workspace host URL
  
   * - profile
     - String
     - The Databricks workspace profile name
  
   * - resource_path
     - String
     - The workspace resource path
  
   * - root_path
     - String
     - The Databricks workspace root path
  
   * - state_path
     - String
     - The workspace state path
  
  
## variables
A Map that defines the custom variables for the bundle, where each key is the name of the variable, and the value is a Map that defines the variable.
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - <variables-entry-name>
     - Map
     - Item of the `variables` map
  
Each item has the following attributes:
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - default
     - Any
     - 
  
   * - description
     - String
     - The description of the variable
  
   * - lookup
     - Map
     - The name of the `alert`, `cluster_policy`, `cluster`, `dashboard`, `instance_pool`, `job`, `metastore`, `pipeline`, `query`, `service_principal`, or `warehouse` object for which to retrieve an ID."
  
   * - type
     - String
     - The type of the variable.
  
  
### variables.lookup
The name of the alert, cluster_policy, cluster, dashboard, instance_pool, job, metastore, pipeline, query, service_principal, or warehouse object for which to retrieve an ID.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - alert
     - String
     - 
  
   * - cluster
     - String
     - 
  
   * - cluster_policy
     - String
     - 
  
   * - dashboard
     - String
     - 
  
   * - instance_pool
     - String
     - 
  
   * - job
     - String
     - 
  
   * - metastore
     - String
     - 
  
   * - notification_destination
     - String
     - 
  
   * - pipeline
     - String
     - 
  
   * - query
     - String
     - 
  
   * - service_principal
     - String
     - 
  
   * - warehouse
     - String
     - 
  
  
## workspace
Defines the Databricks workspace for the bundle.
  
#### Attributes
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - artifact_path
     - String
     - The artifact path to use within the workspace for both deployments and workflow runs
  
   * - auth_type
     - String
     - The authentication type.
  
   * - azure_client_id
     - String
     - The Azure client ID
  
   * - azure_environment
     - String
     - The Azure environment
  
   * - azure_login_app_id
     - String
     - The Azure login app ID
  
   * - azure_tenant_id
     - String
     - The Azure tenant ID
  
   * - azure_use_msi
     - Boolean
     - Whether to use MSI for Azure
  
   * - azure_workspace_resource_id
     - String
     - The Azure workspace resource ID
  
   * - client_id
     - String
     - The client ID for the workspace
  
   * - file_path
     - String
     - The file path to use within the workspace for both deployments and workflow runs
  
   * - google_service_account
     - String
     - The Google service account name
  
   * - host
     - String
     - The Databricks workspace host URL
  
   * - profile
     - String
     - The Databricks workspace profile name
  
   * - resource_path
     - String
     - The workspace resource path
  
   * - root_path
     - String
     - The Databricks workspace root path
  
   * - state_path
     - String
     - The workspace state path
  