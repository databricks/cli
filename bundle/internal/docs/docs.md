  
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
  
  
## run_as
  
The identity to use to run the bundle.
  
  
  
.. list-table::
   :header-rows: 1
  
   * - Key
     - Type
     - Description
  
   * - service_principal_name
     - String
     - 
  
   * - user_name
     - String
     - 
  
  
## sync
  
The files and file paths to include or exclude in the bundle. See [_](/dev-tools/bundles/)
  
  
  
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
  
  
### targets.bundle
  
The name of the bundle when deploying to this target.
  
  
  
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
  
  
### targets.sync
  
The local paths to sync to the target workspace when a bundle is run or deployed.
  
  
  
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
  
  
### targets.workspace
  
The Databricks workspace for the target.
  
  
  
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
  