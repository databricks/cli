# Available `bricks` commands

- [bricks alerts - The alerts API can be used to perform CRUD operations on alerts.](#bricks-alerts---the-alerts-api-can-be-used-to-perform-crud-operations-on-alerts)
    - [bricks alerts create - Create an alert.](#bricks-alerts-create---create-an-alert)
    - [bricks alerts delete - Delete an alert.](#bricks-alerts-delete---delete-an-alert)
    - [bricks alerts get - Get an alert.](#bricks-alerts-get---get-an-alert)
    - [bricks alerts list - Get alerts.](#bricks-alerts-list---get-alerts)
    - [bricks alerts update - Update an alert.](#bricks-alerts-update---update-an-alert)
- [bricks catalogs - A catalog is the first layer of Unity Catalog’s three-level namespace.](#bricks-catalogs---a-catalog-is-the-first-layer-of-unity-catalogs-three-level-namespace)
    - [bricks catalogs create - Create a catalog.](#bricks-catalogs-create---create-a-catalog)
    - [bricks catalogs delete - Delete a catalog.](#bricks-catalogs-delete---delete-a-catalog)
    - [bricks catalogs get - Get a catalog.](#bricks-catalogs-get---get-a-catalog)
    - [bricks catalogs list - List catalogs.](#bricks-catalogs-list---list-catalogs)
    - [bricks catalogs update - Update a catalog.](#bricks-catalogs-update---update-a-catalog)
- [bricks cluster-policies - Cluster policy limits the ability to configure clusters based on a set of rules.](#bricks-cluster-policies---cluster-policy-limits-the-ability-to-configure-clusters-based-on-a-set-of-rules)
    - [bricks cluster-policies create - Create a new policy.](#bricks-cluster-policies-create---create-a-new-policy)
    - [bricks cluster-policies delete - Delete a cluster policy.](#bricks-cluster-policies-delete---delete-a-cluster-policy)
    - [bricks cluster-policies edit - Update a cluster policy.](#bricks-cluster-policies-edit---update-a-cluster-policy)
    - [bricks cluster-policies get - Get entity.](#bricks-cluster-policies-get---get-entity)
    - [bricks cluster-policies list - Get a cluster policy.](#bricks-cluster-policies-list---get-a-cluster-policy)
- [bricks clusters - The Clusters API allows you to create, start, edit, list, terminate, and delete clusters.](#bricks-clusters---the-clusters-api-allows-you-to-create-start-edit-list-terminate-and-delete-clusters)
    - [bricks clusters change-owner - Change cluster owner.](#bricks-clusters-change-owner---change-cluster-owner)
    - [bricks clusters create - Create new cluster.](#bricks-clusters-create---create-new-cluster)
    - [bricks clusters delete - Terminate cluster.](#bricks-clusters-delete---terminate-cluster)
    - [bricks clusters edit - Update cluster configuration.](#bricks-clusters-edit---update-cluster-configuration)
    - [bricks clusters events - List cluster activity events.](#bricks-clusters-events---list-cluster-activity-events)
    - [bricks clusters get - Get cluster info.](#bricks-clusters-get---get-cluster-info)
    - [bricks clusters list - List all clusters.](#bricks-clusters-list---list-all-clusters)
    - [bricks clusters list-node-types - List node types.](#bricks-clusters-list-node-types---list-node-types)
    - [bricks clusters list-zones - List availability zones.](#bricks-clusters-list-zones---list-availability-zones)
    - [bricks clusters permanent-delete - Permanently delete cluster.](#bricks-clusters-permanent-delete---permanently-delete-cluster)
    - [bricks clusters pin - Pin cluster.](#bricks-clusters-pin---pin-cluster)
    - [bricks clusters resize - Resize cluster.](#bricks-clusters-resize---resize-cluster)
    - [bricks clusters restart - Restart cluster.](#bricks-clusters-restart---restart-cluster)
    - [bricks clusters spark-versions - List available Spark versions.](#bricks-clusters-spark-versions---list-available-spark-versions)
    - [bricks clusters start - Start terminated cluster.](#bricks-clusters-start---start-terminated-cluster)
    - [bricks clusters unpin - Unpin cluster.](#bricks-clusters-unpin---unpin-cluster)
- [bricks account credentials - These commands manage credential configurations for this workspace.](#bricks-account-credentials---these-commands-manage-credential-configurations-for-this-workspace)
    - [bricks account credentials create - Create credential configuration.](#bricks-account-credentials-create---create-credential-configuration)
    - [bricks account credentials delete - Delete credential configuration.](#bricks-account-credentials-delete---delete-credential-configuration)
    - [bricks account credentials get - Get credential configuration.](#bricks-account-credentials-get---get-credential-configuration)
    - [bricks account credentials list - Get all credential configurations.](#bricks-account-credentials-list---get-all-credential-configurations)
- [bricks current-user - command allows retrieving information about currently authenticated user or service principal.](#bricks-current-user---command-allows-retrieving-information-about-currently-authenticated-user-or-service-principal)
    - [bricks current-user me - Get current user info.](#bricks-current-user-me---get-current-user-info)
- [bricks account custom-app-integration - manage custom oauth app integrations.](#bricks-account-custom-app-integration---manage-custom-oauth-app-integrations)
    - [bricks account custom-app-integration create - Create Custom OAuth App Integration.](#bricks-account-custom-app-integration-create---create-custom-oauth-app-integration)
    - [bricks account custom-app-integration delete - Delete Custom OAuth App Integration.](#bricks-account-custom-app-integration-delete---delete-custom-oauth-app-integration)
    - [bricks account custom-app-integration get - Get OAuth Custom App Integration.](#bricks-account-custom-app-integration-get---get-oauth-custom-app-integration)
    - [bricks account custom-app-integration list - Get custom oauth app integrations.](#bricks-account-custom-app-integration-list---get-custom-oauth-app-integrations)
    - [bricks account custom-app-integration update - Updates Custom OAuth App Integration.](#bricks-account-custom-app-integration-update---updates-custom-oauth-app-integration)
- [bricks dashboards - Databricks SQL Dashboards](#bricks-dashboards---databricks-sql-dashboards)
    - [bricks dashboards create - Create a dashboard object.](#bricks-dashboards-create---create-a-dashboard-object)
    - [bricks dashboards delete - Remove a dashboard.](#bricks-dashboards-delete---remove-a-dashboard)
    - [bricks dashboards get - Retrieve a definition.](#bricks-dashboards-get---retrieve-a-definition)
    - [bricks dashboards list - Get dashboard objects.](#bricks-dashboards-list---get-dashboard-objects)
    - [bricks dashboards restore - Restore a dashboard.](#bricks-dashboards-restore---restore-a-dashboard)
- [bricks data-sources - command is provided to assist you in making new query objects.](#bricks-data-sources---command-is-provided-to-assist-you-in-making-new-query-objects)
    - [bricks data-sources list - Get a list of SQL warehouses.](#bricks-data-sources-list---get-a-list-of-sql-warehouses)
- [bricks account encryption-keys - manage encryption key configurations.](#bricks-account-encryption-keys---manage-encryption-key-configurations)
    - [bricks account encryption-keys create - Create encryption key configuration.](#bricks-account-encryption-keys-create---create-encryption-key-configuration)
    - [bricks account encryption-keys delete - Delete encryption key configuration.](#bricks-account-encryption-keys-delete---delete-encryption-key-configuration)
    - [bricks account encryption-keys get - Get encryption key configuration.](#bricks-account-encryption-keys-get---get-encryption-key-configuration)
    - [bricks account encryption-keys list - Get all encryption key configurations.](#bricks-account-encryption-keys-list---get-all-encryption-key-configurations)
- [bricks experiments - Manage MLflow experiments](#bricks-experiments---manage-mlflow-experiments)
    - [bricks experiments create-experiment - Create experiment.](#bricks-experiments-create-experiment---create-experiment)
    - [bricks experiments create-run - Create a run.](#bricks-experiments-create-run---create-a-run)
    - [bricks experiments delete-experiment - Delete an experiment.](#bricks-experiments-delete-experiment---delete-an-experiment)
    - [bricks experiments delete-run - Delete a run.](#bricks-experiments-delete-run---delete-a-run)
    - [bricks experiments delete-tag - Delete a tag.](#bricks-experiments-delete-tag---delete-a-tag)
    - [bricks experiments get-by-name - Get metadata.](#bricks-experiments-get-by-name---get-metadata)
    - [bricks experiments get-experiment - Get an experiment.](#bricks-experiments-get-experiment---get-an-experiment)
    - [bricks experiments get-history - Get history of a given metric within a run.](#bricks-experiments-get-history---get-history-of-a-given-metric-within-a-run)
    - [bricks experiments get-run - Get a run.](#bricks-experiments-get-run---get-a-run)
    - [bricks experiments list-artifacts - Get all artifacts.](#bricks-experiments-list-artifacts---get-all-artifacts)
    - [bricks experiments list-experiments - List experiments.](#bricks-experiments-list-experiments---list-experiments)
    - [bricks experiments log-batch - Log a batch.](#bricks-experiments-log-batch---log-a-batch)
    - [bricks experiments log-metric - Log a metric.](#bricks-experiments-log-metric---log-a-metric)
    - [bricks experiments log-model - Log a model.](#bricks-experiments-log-model---log-a-model)
    - [bricks experiments log-param - Log a param.](#bricks-experiments-log-param---log-a-param)
    - [bricks experiments restore-experiment - Restores an experiment.](#bricks-experiments-restore-experiment---restores-an-experiment)
    - [bricks experiments restore-run - Restore a run.](#bricks-experiments-restore-run---restore-a-run)
    - [bricks experiments search-experiments - Search experiments.](#bricks-experiments-search-experiments---search-experiments)
    - [bricks experiments search-runs - Search for runs.](#bricks-experiments-search-runs---search-for-runs)
    - [bricks experiments set-experiment-tag - Set a tag.](#bricks-experiments-set-experiment-tag---set-a-tag)
    - [bricks experiments set-tag - Set a tag.](#bricks-experiments-set-tag---set-a-tag)
    - [bricks experiments update-experiment - Update an experiment.](#bricks-experiments-update-experiment---update-an-experiment)
    - [bricks experiments update-run - Update a run.](#bricks-experiments-update-run---update-a-run)
- [bricks external-locations - manage cloud storage path with a storage credential that authorizes access to it.](#bricks-external-locations---manage-cloud-storage-path-with-a-storage-credential-that-authorizes-access-to-it)
    - [bricks external-locations create - Create an external location.](#bricks-external-locations-create---create-an-external-location)
    - [bricks external-locations delete - Delete an external location.](#bricks-external-locations-delete---delete-an-external-location)
    - [bricks external-locations get - Get an external location.](#bricks-external-locations-get---get-an-external-location)
    - [bricks external-locations list - List external locations.](#bricks-external-locations-list---list-external-locations)
    - [bricks external-locations update - Update an external location.](#bricks-external-locations-update---update-an-external-location)
- [bricks functions - Functions implement User-Defined Functions UDFs in Unity Catalog.](#bricks-functions---functions-implement-user-defined-functions-udfs-in-unity-catalog)
    - [bricks functions create - Create a function.](#bricks-functions-create---create-a-function)
    - [bricks functions delete - Delete a function.](#bricks-functions-delete---delete-a-function)
    - [bricks functions get - Get a function.](#bricks-functions-get---get-a-function)
    - [bricks functions list - List functions.](#bricks-functions-list---list-functions)
    - [bricks functions update - Update a function.](#bricks-functions-update---update-a-function)
- [bricks git-credentials - Registers personal access token for Databricks to do operations on behalf of the user.](#bricks-git-credentials---registers-personal-access-token-for-databricks-to-do-operations-on-behalf-of-the-user)
    - [bricks git-credentials create - Create a credential entry.](#bricks-git-credentials-create---create-a-credential-entry)
    - [bricks git-credentials delete - Delete a credential.](#bricks-git-credentials-delete---delete-a-credential)
    - [bricks git-credentials get - Get a credential entry.](#bricks-git-credentials-get---get-a-credential-entry)
    - [bricks git-credentials list - Get Git credentials.](#bricks-git-credentials-list---get-git-credentials)
    - [bricks git-credentials update - Update a credential.](#bricks-git-credentials-update---update-a-credential)
- [bricks global-init-scripts - configure global initialization scripts for the workspace.](#bricks-global-init-scripts---configure-global-initialization-scripts-for-the-workspace)
    - [bricks global-init-scripts create - Create init script.](#bricks-global-init-scripts-create---create-init-script)
    - [bricks global-init-scripts delete - Delete init script.](#bricks-global-init-scripts-delete---delete-init-script)
    - [bricks global-init-scripts get - Get an init script.](#bricks-global-init-scripts-get---get-an-init-script)
    - [bricks global-init-scripts list - Get init scripts.](#bricks-global-init-scripts-list---get-init-scripts)
    - [bricks global-init-scripts update - Update init script.](#bricks-global-init-scripts-update---update-init-script)
- [bricks grants - Manage data access in Unity Catalog.](#bricks-grants---manage-data-access-in-unity-catalog)
    - [bricks grants get - Get permissions.](#bricks-grants-get---get-permissions)
    - [bricks grants get-effective - Get effective permissions.](#bricks-grants-get-effective---get-effective-permissions)
    - [bricks grants update - Update permissions.](#bricks-grants-update---update-permissions)
- [bricks groups - Groups for identity management.](#bricks-groups---groups-for-identity-management)
    - [bricks groups create - Create a new group.](#bricks-groups-create---create-a-new-group)
    - [bricks groups delete - Delete a group.](#bricks-groups-delete---delete-a-group)
    - [bricks groups get - Get group details.](#bricks-groups-get---get-group-details)
    - [bricks groups list - List group details.](#bricks-groups-list---list-group-details)
    - [bricks groups patch - Update group details.](#bricks-groups-patch---update-group-details)
    - [bricks groups update - Replace a group.](#bricks-groups-update---replace-a-group)
- [bricks account groups - Account-level group management](#bricks-account-groups---account-level-group-management)
    - [bricks account groups create - Create a new group.](#bricks-account-groups-create---create-a-new-group)
    - [bricks account groups delete - Delete a group.](#bricks-account-groups-delete---delete-a-group)
    - [bricks account groups get - Get group details.](#bricks-account-groups-get---get-group-details)
    - [bricks account groups list - List group details.](#bricks-account-groups-list---list-group-details)
    - [bricks account groups patch - Update group details.](#bricks-account-groups-patch---update-group-details)
    - [bricks account groups update - Replace a group.](#bricks-account-groups-update---replace-a-group)
- [bricks instance-pools - manage ready-to-use cloud instances which reduces a cluster start and auto-scaling times.](#bricks-instance-pools---manage-ready-to-use-cloud-instances-which-reduces-a-cluster-start-and-auto-scaling-times)
    - [bricks instance-pools create - Create a new instance pool.](#bricks-instance-pools-create---create-a-new-instance-pool)
    - [bricks instance-pools delete - Delete an instance pool.](#bricks-instance-pools-delete---delete-an-instance-pool)
    - [bricks instance-pools edit - Edit an existing instance pool.](#bricks-instance-pools-edit---edit-an-existing-instance-pool)
    - [bricks instance-pools get - Get instance pool information.](#bricks-instance-pools-get---get-instance-pool-information)
    - [bricks instance-pools list - List instance pool info.](#bricks-instance-pools-list---list-instance-pool-info)
- [bricks instance-profiles - Manage instance profiles that users can launch clusters with.](#bricks-instance-profiles---manage-instance-profiles-that-users-can-launch-clusters-with)
    - [bricks instance-profiles add - Register an instance profile.](#bricks-instance-profiles-add---register-an-instance-profile)
    - [bricks instance-profiles edit - Edit an instance profile.](#bricks-instance-profiles-edit---edit-an-instance-profile)
    - [bricks instance-profiles list - List available instance profiles.](#bricks-instance-profiles-list---list-available-instance-profiles)
    - [bricks instance-profiles remove - Remove the instance profile.](#bricks-instance-profiles-remove---remove-the-instance-profile)
- [bricks ip-access-lists - enable admins to configure IP access lists.](#bricks-ip-access-lists---enable-admins-to-configure-ip-access-lists)
    - [bricks ip-access-lists create - Create access list.](#bricks-ip-access-lists-create---create-access-list)
    - [bricks ip-access-lists delete - Delete access list.](#bricks-ip-access-lists-delete---delete-access-list)
    - [bricks ip-access-lists get - Get access list.](#bricks-ip-access-lists-get---get-access-list)
    - [bricks ip-access-lists list - Get access lists.](#bricks-ip-access-lists-list---get-access-lists)
    - [bricks ip-access-lists replace - Replace access list.](#bricks-ip-access-lists-replace---replace-access-list)
    - [bricks ip-access-lists update - Update access list.](#bricks-ip-access-lists-update---update-access-list)
- [bricks account ip-access-lists - The Accounts IP Access List API enables account admins to configure IP access lists for access to the account console.](#bricks-account-ip-access-lists---the-accounts-ip-access-list-api-enables-account-admins-to-configure-ip-access-lists-for-access-to-the-account-console)
    - [bricks account ip-access-lists create - Create access list.](#bricks-account-ip-access-lists-create---create-access-list)
    - [bricks account ip-access-lists delete - Delete access list.](#bricks-account-ip-access-lists-delete---delete-access-list)
    - [bricks account ip-access-lists get - Get IP access list.](#bricks-account-ip-access-lists-get---get-ip-access-list)
    - [bricks account ip-access-lists list - Get access lists.](#bricks-account-ip-access-lists-list---get-access-lists)
    - [bricks account ip-access-lists replace - Replace access list.](#bricks-account-ip-access-lists-replace---replace-access-list)
    - [bricks account ip-access-lists update - Update access list.](#bricks-account-ip-access-lists-update---update-access-list)
- [bricks jobs - Manage Databricks Workflows.](#bricks-jobs---manage-databricks-workflows)
    - [bricks jobs cancel-all-runs - Cancel all runs of a job.](#bricks-jobs-cancel-all-runs---cancel-all-runs-of-a-job)
    - [bricks jobs cancel-run - Cancel a job run.](#bricks-jobs-cancel-run---cancel-a-job-run)
    - [bricks jobs create - Create a new job.](#bricks-jobs-create---create-a-new-job)
    - [bricks jobs delete - Delete a job.](#bricks-jobs-delete---delete-a-job)
    - [bricks jobs delete-run - Delete a job run.](#bricks-jobs-delete-run---delete-a-job-run)
    - [bricks jobs export-run - Export and retrieve a job run.](#bricks-jobs-export-run---export-and-retrieve-a-job-run)
    - [bricks jobs get - Get a single job.](#bricks-jobs-get---get-a-single-job)
    - [bricks jobs get-run - Get a single job run.](#bricks-jobs-get-run---get-a-single-job-run)
    - [bricks jobs get-run-output - Get the output for a single run.](#bricks-jobs-get-run-output---get-the-output-for-a-single-run)
    - [bricks jobs list - List all jobs.](#bricks-jobs-list---list-all-jobs)
    - [bricks jobs list-runs - List runs for a job.](#bricks-jobs-list-runs---list-runs-for-a-job)
    - [bricks jobs repair-run - Repair a job run.](#bricks-jobs-repair-run---repair-a-job-run)
    - [bricks jobs reset - Overwrites all settings for a job.](#bricks-jobs-reset---overwrites-all-settings-for-a-job)
    - [bricks jobs run-now - Trigger a new job run.](#bricks-jobs-run-now---trigger-a-new-job-run)
    - [bricks jobs submit - Create and trigger a one-time run.](#bricks-jobs-submit---create-and-trigger-a-one-time-run)
    - [bricks jobs update - Partially updates a job.](#bricks-jobs-update---partially-updates-a-job)
- [bricks libraries - Manage libraries on a cluster.](#bricks-libraries---manage-libraries-on-a-cluster)
    - [bricks libraries all-cluster-statuses - Get all statuses.](#bricks-libraries-all-cluster-statuses---get-all-statuses)
    - [bricks libraries cluster-status - Get status.](#bricks-libraries-cluster-status---get-status)
    - [bricks libraries install - Add a library.](#bricks-libraries-install---add-a-library)
    - [bricks libraries uninstall - Uninstall libraries.](#bricks-libraries-uninstall---uninstall-libraries)
- [bricks account log-delivery - These commands manage log delivery configurations for this account.](#bricks-account-log-delivery---these-commands-manage-log-delivery-configurations-for-this-account)
    - [bricks account log-delivery create - Create a new log delivery configuration.](#bricks-account-log-delivery-create---create-a-new-log-delivery-configuration)
    - [bricks account log-delivery get - Get log delivery configuration.](#bricks-account-log-delivery-get---get-log-delivery-configuration)
    - [bricks account log-delivery list - Get all log delivery configurations.](#bricks-account-log-delivery-list---get-all-log-delivery-configurations)
    - [bricks account log-delivery patch-status - Enable or disable log delivery configuration.](#bricks-account-log-delivery-patch-status---enable-or-disable-log-delivery-configuration)
- [bricks account metastore-assignments - These commands manage metastore assignments to a workspace.](#bricks-account-metastore-assignments---these-commands-manage-metastore-assignments-to-a-workspace)
    - [bricks account metastore-assignments create - Assigns a workspace to a metastore.](#bricks-account-metastore-assignments-create---assigns-a-workspace-to-a-metastore)
    - [bricks account metastore-assignments delete - Delete a metastore assignment.](#bricks-account-metastore-assignments-delete---delete-a-metastore-assignment)
    - [bricks account metastore-assignments get - Gets the metastore assignment for a workspace.](#bricks-account-metastore-assignments-get---gets-the-metastore-assignment-for-a-workspace)
    - [bricks account metastore-assignments list - Get all workspaces assigned to a metastore.](#bricks-account-metastore-assignments-list---get-all-workspaces-assigned-to-a-metastore)
    - [bricks account metastore-assignments update - Updates a metastore assignment to a workspaces.](#bricks-account-metastore-assignments-update---updates-a-metastore-assignment-to-a-workspaces)
- [bricks metastores - Manage metastores in Unity Catalog.](#bricks-metastores---manage-metastores-in-unity-catalog)
    - [bricks metastores assign - Create an assignment.](#bricks-metastores-assign---create-an-assignment)
    - [bricks metastores create - Create a metastore.](#bricks-metastores-create---create-a-metastore)
    - [bricks metastores current - Get metastore assignment for workspace.](#bricks-metastores-current---get-metastore-assignment-for-workspace)
    - [bricks metastores delete - Delete a metastore.](#bricks-metastores-delete---delete-a-metastore)
    - [bricks metastores get - Get a metastore.](#bricks-metastores-get---get-a-metastore)
    - [bricks metastores list - List metastores.](#bricks-metastores-list---list-metastores)
    - [bricks metastores maintenance - Enables or disables auto maintenance on the metastore.](#bricks-metastores-maintenance---enables-or-disables-auto-maintenance-on-the-metastore)
    - [bricks metastores summary - Get a metastore summary.](#bricks-metastores-summary---get-a-metastore-summary)
    - [bricks metastores unassign - Delete an assignment.](#bricks-metastores-unassign---delete-an-assignment)
    - [bricks metastores update - Update a metastore.](#bricks-metastores-update---update-a-metastore)
    - [bricks metastores update-assignment - Update an assignment.](#bricks-metastores-update-assignment---update-an-assignment)
- [bricks account metastores - These commands manage Unity Catalog metastores for an account.](#bricks-account-metastores---these-commands-manage-unity-catalog-metastores-for-an-account)
    - [bricks account metastores create - Create metastore.](#bricks-account-metastores-create---create-metastore)
    - [bricks account metastores delete - Delete a metastore.](#bricks-account-metastores-delete---delete-a-metastore)
    - [bricks account metastores get - Get a metastore.](#bricks-account-metastores-get---get-a-metastore)
    - [bricks account metastores list - Get all metastores associated with an account.](#bricks-account-metastores-list---get-all-metastores-associated-with-an-account)
    - [bricks account metastores update - Update a metastore.](#bricks-account-metastores-update---update-a-metastore)
- [bricks model-registry - Expose commands for Model Registry.](#bricks-model-registry---expose-commands-for-model-registry)
    - [bricks model-registry approve-transition-request - Approve transition request.](#bricks-model-registry-approve-transition-request---approve-transition-request)
    - [bricks model-registry create-comment - Post a comment.](#bricks-model-registry-create-comment---post-a-comment)
    - [bricks model-registry create-model - Create a model.](#bricks-model-registry-create-model---create-a-model)
    - [bricks model-registry create-model-version - Create a model version.](#bricks-model-registry-create-model-version---create-a-model-version)
    - [bricks model-registry create-transition-request - Make a transition request.](#bricks-model-registry-create-transition-request---make-a-transition-request)
    - [bricks model-registry create-webhook - Create a webhook.](#bricks-model-registry-create-webhook---create-a-webhook)
    - [bricks model-registry delete-comment - Delete a comment.](#bricks-model-registry-delete-comment---delete-a-comment)
    - [bricks model-registry delete-model - Delete a model.](#bricks-model-registry-delete-model---delete-a-model)
    - [bricks model-registry delete-model-tag - Delete a model tag.](#bricks-model-registry-delete-model-tag---delete-a-model-tag)
    - [bricks model-registry delete-model-version - Delete a model version.](#bricks-model-registry-delete-model-version---delete-a-model-version)
    - [bricks model-registry delete-model-version-tag - Delete a model version tag.](#bricks-model-registry-delete-model-version-tag---delete-a-model-version-tag)
    - [bricks model-registry delete-transition-request - Delete a ransition request.](#bricks-model-registry-delete-transition-request---delete-a-ransition-request)
    - [bricks model-registry delete-webhook - Delete a webhook.](#bricks-model-registry-delete-webhook---delete-a-webhook)
    - [bricks model-registry get-latest-versions - Get the latest version.](#bricks-model-registry-get-latest-versions---get-the-latest-version)
    - [bricks model-registry get-model - Get model.](#bricks-model-registry-get-model---get-model)
    - [bricks model-registry get-model-version - Get a model version.](#bricks-model-registry-get-model-version---get-a-model-version)
    - [bricks model-registry get-model-version-download-uri - Get a model version URI.](#bricks-model-registry-get-model-version-download-uri---get-a-model-version-uri)
    - [bricks model-registry list-models - List models.](#bricks-model-registry-list-models---list-models)
    - [bricks model-registry list-transition-requests - List transition requests.](#bricks-model-registry-list-transition-requests---list-transition-requests)
    - [bricks model-registry list-webhooks - List registry webhooks.](#bricks-model-registry-list-webhooks---list-registry-webhooks)
    - [bricks model-registry reject-transition-request - Reject a transition request.](#bricks-model-registry-reject-transition-request---reject-a-transition-request)
    - [bricks model-registry rename-model - Rename a model.](#bricks-model-registry-rename-model---rename-a-model)
    - [bricks model-registry search-model-versions - Searches model versions.](#bricks-model-registry-search-model-versions---searches-model-versions)
    - [bricks model-registry search-models - Search models.](#bricks-model-registry-search-models---search-models)
    - [bricks model-registry set-model-tag - Set a tag.](#bricks-model-registry-set-model-tag---set-a-tag)
    - [bricks model-registry set-model-version-tag - Set a version tag.](#bricks-model-registry-set-model-version-tag---set-a-version-tag)
    - [bricks model-registry test-registry-webhook - Test a webhook.](#bricks-model-registry-test-registry-webhook---test-a-webhook)
    - [bricks model-registry transition-stage - Transition a stage.](#bricks-model-registry-transition-stage---transition-a-stage)
    - [bricks model-registry update-comment - Update a comment.](#bricks-model-registry-update-comment---update-a-comment)
    - [bricks model-registry update-model - Update model.](#bricks-model-registry-update-model---update-model)
    - [bricks model-registry update-model-version - Update model version.](#bricks-model-registry-update-model-version---update-model-version)
    - [bricks model-registry update-webhook - Update a webhook.](#bricks-model-registry-update-webhook---update-a-webhook)
- [bricks account networks - Manage network configurations.](#bricks-account-networks---manage-network-configurations)
    - [bricks account networks create - Create network configuration.](#bricks-account-networks-create---create-network-configuration)
    - [bricks account networks delete - Delete a network configuration.](#bricks-account-networks-delete---delete-a-network-configuration)
    - [bricks account networks get - Get a network configuration.](#bricks-account-networks-get---get-a-network-configuration)
    - [bricks account networks list - Get all network configurations.](#bricks-account-networks-list---get-all-network-configurations)
- [bricks account o-auth-enrollment - These commands enable administrators to enroll OAuth for their accounts, which is required for adding/using any OAuth published/custom application integration.](#bricks-account-o-auth-enrollment---these-commands-enable-administrators-to-enroll-oauth-for-their-accounts-which-is-required-for-addingusing-any-oauth-publishedcustom-application-integration)
    - [bricks account o-auth-enrollment create - Create OAuth Enrollment request.](#bricks-account-o-auth-enrollment-create---create-oauth-enrollment-request)
    - [bricks account o-auth-enrollment get - Get OAuth enrollment status.](#bricks-account-o-auth-enrollment-get---get-oauth-enrollment-status)
- [bricks permissions - Manage access for various users on different objects and endpoints.](#bricks-permissions---manage-access-for-various-users-on-different-objects-and-endpoints)
    - [bricks permissions get - Get object permissions.](#bricks-permissions-get---get-object-permissions)
    - [bricks permissions get-permission-levels - Get permission levels.](#bricks-permissions-get-permission-levels---get-permission-levels)
    - [bricks permissions set - Set permissions.](#bricks-permissions-set---set-permissions)
    - [bricks permissions update - Update permission.](#bricks-permissions-update---update-permission)
- [bricks pipelines - Manage Delta Live Tables from command-line.](#bricks-pipelines---manage-delta-live-tables-from-command-line)
    - [bricks pipelines create - Create a pipeline.](#bricks-pipelines-create---create-a-pipeline)
    - [bricks pipelines delete - Delete a pipeline.](#bricks-pipelines-delete---delete-a-pipeline)
    - [bricks pipelines get - Get a pipeline.](#bricks-pipelines-get---get-a-pipeline)
    - [bricks pipelines get-update - Get a pipeline update.](#bricks-pipelines-get-update---get-a-pipeline-update)
    - [bricks pipelines list-pipeline-events - List pipeline events.](#bricks-pipelines-list-pipeline-events---list-pipeline-events)
    - [bricks pipelines list-pipelines - List pipelines.](#bricks-pipelines-list-pipelines---list-pipelines)
    - [bricks pipelines list-updates - List pipeline updates.](#bricks-pipelines-list-updates---list-pipeline-updates)
    - [bricks pipelines reset - Reset a pipeline.](#bricks-pipelines-reset---reset-a-pipeline)
    - [bricks pipelines start-update - Queue a pipeline update.](#bricks-pipelines-start-update---queue-a-pipeline-update)
    - [bricks pipelines stop - Stop a pipeline.](#bricks-pipelines-stop---stop-a-pipeline)
    - [bricks pipelines update - Edit a pipeline.](#bricks-pipelines-update---edit-a-pipeline)
- [bricks policy-families - View available policy families.](#bricks-policy-families---view-available-policy-families)
    - [bricks policy-families get - get cluster policy family.](#bricks-policy-families-get---get-cluster-policy-family)
    - [bricks policy-families list - list policy families.](#bricks-policy-families-list---list-policy-families)
- [bricks account private-access - PrivateLink settings.](#bricks-account-private-access---privatelink-settings)
    - [bricks account private-access create - Create private access settings.](#bricks-account-private-access-create---create-private-access-settings)
    - [bricks account private-access delete - Delete a private access settings object.](#bricks-account-private-access-delete---delete-a-private-access-settings-object)
    - [bricks account private-access get - Get a private access settings object.](#bricks-account-private-access-get---get-a-private-access-settings-object)
    - [bricks account private-access list - Get all private access settings objects.](#bricks-account-private-access-list---get-all-private-access-settings-objects)
    - [bricks account private-access replace - Replace private access settings.](#bricks-account-private-access-replace---replace-private-access-settings)
- [bricks providers - Delta Sharing Providers commands.](#bricks-providers---delta-sharing-providers-commands)
    - [bricks providers create - Create an auth provider.](#bricks-providers-create---create-an-auth-provider)
    - [bricks providers delete - Delete a provider.](#bricks-providers-delete---delete-a-provider)
    - [bricks providers get - Get a provider.](#bricks-providers-get---get-a-provider)
    - [bricks providers list - List providers.](#bricks-providers-list---list-providers)
    - [bricks providers list-shares - List shares by Provider.](#bricks-providers-list-shares---list-shares-by-provider)
    - [bricks providers update - Update a provider.](#bricks-providers-update---update-a-provider)
- [bricks account published-app-integration - manage published OAuth app integrations like Tableau Cloud for Databricks in AWS cloud.](#bricks-account-published-app-integration---manage-published-oauth-app-integrations-like-tableau-cloud-for-databricks-in-aws-cloud)
    - [bricks account published-app-integration create - Create Published OAuth App Integration.](#bricks-account-published-app-integration-create---create-published-oauth-app-integration)
    - [bricks account published-app-integration delete - Delete Published OAuth App Integration.](#bricks-account-published-app-integration-delete---delete-published-oauth-app-integration)
    - [bricks account published-app-integration get - Get OAuth Published App Integration.](#bricks-account-published-app-integration-get---get-oauth-published-app-integration)
    - [bricks account published-app-integration list - Get published oauth app integrations.](#bricks-account-published-app-integration-list---get-published-oauth-app-integrations)
    - [bricks account published-app-integration update - Updates Published OAuth App Integration.](#bricks-account-published-app-integration-update---updates-published-oauth-app-integration)
- [bricks queries - These endpoints are used for CRUD operations on query definitions.](#bricks-queries---these-endpoints-are-used-for-crud-operations-on-query-definitions)
    - [bricks queries create - Create a new query definition.](#bricks-queries-create---create-a-new-query-definition)
    - [bricks queries delete - Delete a query.](#bricks-queries-delete---delete-a-query)
    - [bricks queries get - Get a query definition.](#bricks-queries-get---get-a-query-definition)
    - [bricks queries list - Get a list of queries.](#bricks-queries-list---get-a-list-of-queries)
    - [bricks queries restore - Restore a query.](#bricks-queries-restore---restore-a-query)
    - [bricks queries update - Change a query definition.](#bricks-queries-update---change-a-query-definition)
- [bricks query-history - Access the history of queries through SQL warehouses.](#bricks-query-history---access-the-history-of-queries-through-sql-warehouses)
    - [bricks query-history list - List Queries.](#bricks-query-history-list---list-queries)
- [bricks recipient-activation - Delta Sharing recipient activation commands.](#bricks-recipient-activation---delta-sharing-recipient-activation-commands)
    - [bricks recipient-activation get-activation-url-info - Get a share activation URL.](#bricks-recipient-activation-get-activation-url-info---get-a-share-activation-url)
    - [bricks recipient-activation retrieve-token - Get an access token.](#bricks-recipient-activation-retrieve-token---get-an-access-token)
- [bricks recipients - Delta Sharing recipients.](#bricks-recipients---delta-sharing-recipients)
    - [bricks recipients create - Create a share recipient.](#bricks-recipients-create---create-a-share-recipient)
    - [bricks recipients delete - Delete a share recipient.](#bricks-recipients-delete---delete-a-share-recipient)
    - [bricks recipients get - Get a share recipient.](#bricks-recipients-get---get-a-share-recipient)
    - [bricks recipients list - List share recipients.](#bricks-recipients-list---list-share-recipients)
    - [bricks recipients rotate-token - Rotate a token.](#bricks-recipients-rotate-token---rotate-a-token)
    - [bricks recipients share-permissions - Get recipient share permissions.](#bricks-recipients-share-permissions---get-recipient-share-permissions)
    - [bricks recipients update - Update a share recipient.](#bricks-recipients-update---update-a-share-recipient)
- [bricks repos - Manage their git repos.](#bricks-repos---manage-their-git-repos)
    - [bricks repos create - Create a repo.](#bricks-repos-create---create-a-repo)
    - [bricks repos delete - Delete a repo.](#bricks-repos-delete---delete-a-repo)
    - [bricks repos get - Get a repo.](#bricks-repos-get---get-a-repo)
    - [bricks repos list - Get repos.](#bricks-repos-list---get-repos)
    - [bricks repos update - Update a repo.](#bricks-repos-update---update-a-repo)
- [bricks schemas - Manage schemas in Unity Catalog.](#bricks-schemas---manage-schemas-in-unity-catalog)
    - [bricks schemas create - Create a schema.](#bricks-schemas-create---create-a-schema)
    - [bricks schemas delete - Delete a schema.](#bricks-schemas-delete---delete-a-schema)
    - [bricks schemas get - Get a schema.](#bricks-schemas-get---get-a-schema)
    - [bricks schemas list - List schemas.](#bricks-schemas-list---list-schemas)
    - [bricks schemas update - Update a schema.](#bricks-schemas-update---update-a-schema)
- [bricks secrets - manage secrets, secret scopes, and access permissions.](#bricks-secrets---manage-secrets-secret-scopes-and-access-permissions)
    - [bricks secrets create-scope - Create a new secret scope.](#bricks-secrets-create-scope---create-a-new-secret-scope)
    - [bricks secrets delete-acl - Delete an ACL.](#bricks-secrets-delete-acl---delete-an-acl)
    - [bricks secrets delete-scope - Delete a secret scope.](#bricks-secrets-delete-scope---delete-a-secret-scope)
    - [bricks secrets delete-secret - Delete a secret.](#bricks-secrets-delete-secret---delete-a-secret)
    - [bricks secrets get-acl - Get secret ACL details.](#bricks-secrets-get-acl---get-secret-acl-details)
    - [bricks secrets list-acls - Lists ACLs.](#bricks-secrets-list-acls---lists-acls)
    - [bricks secrets list-scopes - List all scopes.](#bricks-secrets-list-scopes---list-all-scopes)
    - [bricks secrets list-secrets - List secret keys.](#bricks-secrets-list-secrets---list-secret-keys)
    - [bricks secrets put-acl - Create/update an ACL.](#bricks-secrets-put-acl---createupdate-an-acl)
    - [bricks secrets put-secret - Add a secret.](#bricks-secrets-put-secret---add-a-secret)
- [bricks service-principals - Manage service principals.](#bricks-service-principals---manage-service-principals)
    - [bricks service-principals create - Create a service principal.](#bricks-service-principals-create---create-a-service-principal)
    - [bricks service-principals delete - Delete a service principal.](#bricks-service-principals-delete---delete-a-service-principal)
    - [bricks service-principals get - Get service principal details.](#bricks-service-principals-get---get-service-principal-details)
    - [bricks service-principals list - List service principals.](#bricks-service-principals-list---list-service-principals)
    - [bricks service-principals patch - Update service principal details.](#bricks-service-principals-patch---update-service-principal-details)
    - [bricks service-principals update - Replace service principal.](#bricks-service-principals-update---replace-service-principal)
- [bricks account service-principals - Manage service principals on the account level.](#bricks-account-service-principals---manage-service-principals-on-the-account-level)
    - [bricks account service-principals create - Create a service principal.](#bricks-account-service-principals-create---create-a-service-principal)
    - [bricks account service-principals delete - Delete a service principal.](#bricks-account-service-principals-delete---delete-a-service-principal)
    - [bricks account service-principals get - Get service principal details.](#bricks-account-service-principals-get---get-service-principal-details)
    - [bricks account service-principals list - List service principals.](#bricks-account-service-principals-list---list-service-principals)
    - [bricks account service-principals patch - Update service principal details.](#bricks-account-service-principals-patch---update-service-principal-details)
    - [bricks account service-principals update - Replace service principal.](#bricks-account-service-principals-update---replace-service-principal)
- [bricks serving-endpoints - Manage model serving endpoints.](#bricks-serving-endpoints---manage-model-serving-endpoints)
    - [bricks serving-endpoints build-logs - Retrieve the logs associated with building the model's environment for a given serving endpoint's served model.](#bricks-serving-endpoints-build-logs---retrieve-the-logs-associated-with-building-the-models-environment-for-a-given-serving-endpoints-served-model)
    - [bricks serving-endpoints create - Create a new serving endpoint.](#bricks-serving-endpoints-create---create-a-new-serving-endpoint)
    - [bricks serving-endpoints delete - Delete a serving endpoint.](#bricks-serving-endpoints-delete---delete-a-serving-endpoint)
    - [bricks serving-endpoints export-metrics - Retrieve the metrics corresponding to a serving endpoint for the current time in Prometheus or OpenMetrics exposition format.](#bricks-serving-endpoints-export-metrics---retrieve-the-metrics-corresponding-to-a-serving-endpoint-for-the-current-time-in-prometheus-or-openmetrics-exposition-format)
    - [bricks serving-endpoints get - Get a single serving endpoint.](#bricks-serving-endpoints-get---get-a-single-serving-endpoint)
    - [bricks serving-endpoints list - Retrieve all serving endpoints.](#bricks-serving-endpoints-list---retrieve-all-serving-endpoints)
    - [bricks serving-endpoints logs - Retrieve the most recent log lines associated with a given serving endpoint's served model.](#bricks-serving-endpoints-logs---retrieve-the-most-recent-log-lines-associated-with-a-given-serving-endpoints-served-model)
    - [bricks serving-endpoints query - Query a serving endpoint with provided model input.](#bricks-serving-endpoints-query---query-a-serving-endpoint-with-provided-model-input)
    - [bricks serving-endpoints update-config - Update a serving endpoint with a new config.](#bricks-serving-endpoints-update-config---update-a-serving-endpoint-with-a-new-config)
- [bricks shares - Databricks Shares commands.](#bricks-shares---databricks-shares-commands)
    - [bricks shares create - Create a share.](#bricks-shares-create---create-a-share)
    - [bricks shares delete - Delete a share.](#bricks-shares-delete---delete-a-share)
    - [bricks shares get - Get a share.](#bricks-shares-get---get-a-share)
    - [bricks shares list - List shares.](#bricks-shares-list---list-shares)
    - [bricks shares share-permissions - Get permissions.](#bricks-shares-share-permissions---get-permissions)
    - [bricks shares update - Update a share.](#bricks-shares-update---update-a-share)
    - [bricks shares update-permissions - Update permissions.](#bricks-shares-update-permissions---update-permissions)
- [bricks account storage - Manage storage configurations for this workspace.](#bricks-account-storage---manage-storage-configurations-for-this-workspace)
    - [bricks account storage create - Create new storage configuration.](#bricks-account-storage-create---create-new-storage-configuration)
    - [bricks account storage delete - Delete storage configuration.](#bricks-account-storage-delete---delete-storage-configuration)
    - [bricks account storage get - Get storage configuration.](#bricks-account-storage-get---get-storage-configuration)
    - [bricks account storage list - Get all storage configurations.](#bricks-account-storage-list---get-all-storage-configurations)
- [bricks storage-credentials - Manage storage credentials for Unity Catalog.](#bricks-storage-credentials---manage-storage-credentials-for-unity-catalog)
    - [bricks storage-credentials create - Create a storage credential.](#bricks-storage-credentials-create---create-a-storage-credential)
    - [bricks storage-credentials delete - Delete a credential.](#bricks-storage-credentials-delete---delete-a-credential)
    - [bricks storage-credentials get - Get a credential.](#bricks-storage-credentials-get---get-a-credential)
    - [bricks storage-credentials list - List credentials.](#bricks-storage-credentials-list---list-credentials)
    - [bricks storage-credentials update - Update a credential.](#bricks-storage-credentials-update---update-a-credential)
    - [bricks storage-credentials validate - Validate a storage credential.](#bricks-storage-credentials-validate---validate-a-storage-credential)
- [bricks account storage-credentials - These commands manage storage credentials for a particular metastore.](#bricks-account-storage-credentials---these-commands-manage-storage-credentials-for-a-particular-metastore)
    - [bricks account storage-credentials create - Create a storage credential.](#bricks-account-storage-credentials-create---create-a-storage-credential)
    - [bricks account storage-credentials get - Gets the named storage credential.](#bricks-account-storage-credentials-get---gets-the-named-storage-credential)
    - [bricks account storage-credentials list - Get all storage credentials assigned to a metastore.](#bricks-account-storage-credentials-list---get-all-storage-credentials-assigned-to-a-metastore)
- [bricks table-constraints - Primary key and foreign key constraints encode relationships between fields in tables.](#bricks-table-constraints---primary-key-and-foreign-key-constraints-encode-relationships-between-fields-in-tables)
    - [bricks table-constraints create - Create a table constraint.](#bricks-table-constraints-create---create-a-table-constraint)
    - [bricks table-constraints delete - Delete a table constraint.](#bricks-table-constraints-delete---delete-a-table-constraint)
- [bricks tables - A table resides in the third layer of Unity Catalog’s three-level namespace.](#bricks-tables---a-table-resides-in-the-third-layer-of-unity-catalogs-three-level-namespace)
    - [bricks tables delete - Delete a table.](#bricks-tables-delete---delete-a-table)
    - [bricks tables get - Get a table.](#bricks-tables-get---get-a-table)
    - [bricks tables list - List tables.](#bricks-tables-list---list-tables)
    - [bricks tables list-summaries - List table summaries.](#bricks-tables-list-summaries---list-table-summaries)
- [bricks token-management - Enables administrators to get all tokens and delete tokens for other users.](#bricks-token-management---enables-administrators-to-get-all-tokens-and-delete-tokens-for-other-users)
    - [bricks token-management create-obo-token - Create on-behalf token.](#bricks-token-management-create-obo-token---create-on-behalf-token)
    - [bricks token-management delete - Delete a token.](#bricks-token-management-delete---delete-a-token)
    - [bricks token-management get - Get token info.](#bricks-token-management-get---get-token-info)
    - [bricks token-management list - List all tokens.](#bricks-token-management-list---list-all-tokens)
- [bricks tokens - The Token API allows you to create, list, and revoke tokens that can be used to authenticate and access Databricks commandss.](#bricks-tokens---the-token-api-allows-you-to-create-list-and-revoke-tokens-that-can-be-used-to-authenticate-and-access-databricks-commandss)
    - [bricks tokens create - Create a user token.](#bricks-tokens-create---create-a-user-token)
    - [bricks tokens delete - Revoke token.](#bricks-tokens-delete---revoke-token)
    - [bricks tokens list - List tokens.](#bricks-tokens-list---list-tokens)
- [bricks users - Manage users on the workspace-level.](#bricks-users---manage-users-on-the-workspace-level)
    - [bricks users create - Create a new user.](#bricks-users-create---create-a-new-user)
    - [bricks users delete - Delete a user.](#bricks-users-delete---delete-a-user)
    - [bricks users get - Get user details.](#bricks-users-get---get-user-details)
    - [bricks users list - List users.](#bricks-users-list---list-users)
    - [bricks users patch - Update user details.](#bricks-users-patch---update-user-details)
    - [bricks users update - Replace a user.](#bricks-users-update---replace-a-user)
- [bricks account users - Manage users on the accou](#bricks-account-users---manage-users-on-the-accou)
    - [bricks account users create - Create a new user.](#bricks-account-users-create---create-a-new-user)
    - [bricks account users delete - Delete a user.](#bricks-account-users-delete---delete-a-user)
    - [bricks account users get - Get user details.](#bricks-account-users-get---get-user-details)
    - [bricks account users list - List users.](#bricks-account-users-list---list-users)
    - [bricks account users patch - Update user details.](#bricks-account-users-patch---update-user-details)
    - [bricks account users update - Replace a user.](#bricks-account-users-update---replace-a-user)
- [bricks account vpc-endpoints - Manage VPC endpoints.](#bricks-account-vpc-endpoints---manage-vpc-endpoints)
    - [bricks account vpc-endpoints create - Create VPC endpoint configuration.](#bricks-account-vpc-endpoints-create---create-vpc-endpoint-configuration)
    - [bricks account vpc-endpoints delete - Delete VPC endpoint configuration.](#bricks-account-vpc-endpoints-delete---delete-vpc-endpoint-configuration)
    - [bricks account vpc-endpoints get - Get a VPC endpoint configuration.](#bricks-account-vpc-endpoints-get---get-a-vpc-endpoint-configuration)
    - [bricks account vpc-endpoints list - Get all VPC endpoint configurations.](#bricks-account-vpc-endpoints-list---get-all-vpc-endpoint-configurations)
- [bricks warehouses - Manage Databricks SQL warehouses.](#bricks-warehouses---manage-databricks-sql-warehouses)
    - [bricks warehouses create - Create a warehouse.](#bricks-warehouses-create---create-a-warehouse)
    - [bricks warehouses delete - Delete a warehouse.](#bricks-warehouses-delete---delete-a-warehouse)
    - [bricks warehouses edit - Update a warehouse.](#bricks-warehouses-edit---update-a-warehouse)
    - [bricks warehouses get - Get warehouse info.](#bricks-warehouses-get---get-warehouse-info)
    - [bricks warehouses get-workspace-warehouse-config - Get the workspace configuration.](#bricks-warehouses-get-workspace-warehouse-config---get-the-workspace-configuration)
    - [bricks warehouses list - List warehouses.](#bricks-warehouses-list---list-warehouses)
    - [bricks warehouses set-workspace-warehouse-config - Set the workspace configuration.](#bricks-warehouses-set-workspace-warehouse-config---set-the-workspace-configuration)
    - [bricks warehouses start - Start a warehouse.](#bricks-warehouses-start---start-a-warehouse)
    - [bricks warehouses stop - Stop a warehouse.](#bricks-warehouses-stop---stop-a-warehouse)
- [bricks workspace - The Workspace API allows you to list, import, export, and delete notebooks and folders.](#bricks-workspace---the-workspace-api-allows-you-to-list-import-export-and-delete-notebooks-and-folders)
    - [bricks workspace delete - Delete a workspace object.](#bricks-workspace-delete---delete-a-workspace-object)
    - [bricks workspace export - Export a workspace object.](#bricks-workspace-export---export-a-workspace-object)
    - [bricks workspace get-status - Get status.](#bricks-workspace-get-status---get-status)
    - [bricks workspace import - Import a workspace object.](#bricks-workspace-import---import-a-workspace-object)
    - [bricks workspace list - List contents.](#bricks-workspace-list---list-contents)
    - [bricks workspace mkdirs - Create a directory.](#bricks-workspace-mkdirs---create-a-directory)
- [bricks account workspace-assignment - The Workspace Permission Assignment API allows you to manage workspace permissions for principals in your account.](#bricks-account-workspace-assignment---the-workspace-permission-assignment-api-allows-you-to-manage-workspace-permissions-for-principals-in-your-account)
    - [bricks account workspace-assignment delete - Delete permissions assignment.](#bricks-account-workspace-assignment-delete---delete-permissions-assignment)
    - [bricks account workspace-assignment get - List workspace permissions.](#bricks-account-workspace-assignment-get---list-workspace-permissions)
    - [bricks account workspace-assignment list - Get permission assignments.](#bricks-account-workspace-assignment-list---get-permission-assignments)
    - [bricks account workspace-assignment update - Create or update permissions assignment.](#bricks-account-workspace-assignment-update---create-or-update-permissions-assignment)
- [bricks workspace-conf - command allows updating known workspace settings for advanced users.](#bricks-workspace-conf---command-allows-updating-known-workspace-settings-for-advanced-users)
    - [bricks workspace-conf get-status - Check configuration status.](#bricks-workspace-conf-get-status---check-configuration-status)
    - [bricks workspace-conf set-status - Enable/disable features.](#bricks-workspace-conf-set-status---enabledisable-features)
- [bricks account workspaces - These commands manage workspaces for this account.](#bricks-account-workspaces---these-commands-manage-workspaces-for-this-account)
    - [bricks account workspaces create - Create a new workspace.](#bricks-account-workspaces-create---create-a-new-workspace)
    - [bricks account workspaces delete - Delete a workspace.](#bricks-account-workspaces-delete---delete-a-workspace)
    - [bricks account workspaces get - Get a workspace.](#bricks-account-workspaces-get---get-a-workspace)
    - [bricks account workspaces list - Get all workspaces.](#bricks-account-workspaces-list---get-all-workspaces)
    - [bricks account workspaces update - Update workspace configuration.](#bricks-account-workspaces-update---update-workspace-configuration)


## `bricks alerts` - The alerts API can be used to perform CRUD operations on alerts.

The alerts API can be used to perform CRUD operations on alerts. An alert is a Databricks SQL
object that periodically runs a query, evaluates a condition of its result, and notifies one
or more users and/or notification destinations if the condition was met.

### `bricks alerts create` - Create an alert.

An alert is a Databricks SQL object that periodically runs a query, evaluates a condition of its result,
and notifies users or notification destinations if the condition was met.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body * `--parent` - The identifier of the workspace folder containing the alert.
 * `--rearm` - Number of seconds after being triggered before the alert rearms itself and can be triggered again.

### `bricks alerts delete` - Delete an alert.

Deletes an alert. Deleted alerts are no longer accessible and cannot be restored.
**Note:** Unlike queries and dashboards, alerts cannot be moved to the trash.

### `bricks alerts get` - Get an alert.

Gets an alert.

### `bricks alerts list` - Get alerts.

Gets a list of alerts.

### `bricks alerts update` - Update an alert.

Updates an alert.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--rearm` - Number of seconds after being triggered before the alert rearms itself and can be triggered again.

## `bricks catalogs` - A catalog is the first layer of Unity Catalog’s three-level namespace.

A catalog is the first layer of Unity Catalog’s three-level namespace. It’s used to organize
your data assets. Users can see all catalogs on which they have been assigned the USE_CATALOG
data permission.

In Unity Catalog, admins and data stewards manage users and their access to data centrally
across all of the workspaces in a Databricks account. Users in different workspaces can
share access to the same data, depending on privileges granted centrally in Unity Catalog.

### `bricks catalogs create` - Create a catalog.

Creates a new catalog instance in the parent metastore if the caller is a metastore admin or has the **CREATE_CATALOG** privilege.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - User-provided free-form text description.
 * `--provider-name` - The name of delta sharing provider.
 * `--share-name` - The name of the share under the share provider.
 * `--storage-root` - Storage root URL for managed tables within catalog.

### `bricks catalogs delete` - Delete a catalog.

Deletes the catalog that matches the supplied name. The caller must be a metastore admin or the owner of the catalog.

Flags:
* `--force` - Force deletion even if the catalog is not empty.

### `bricks catalogs get` - Get a catalog.

Gets the specified catalog in a metastore. The caller must be a metastore admin, the owner of the catalog, or a user that has the **USE_CATALOG** privilege set for their account.

### `bricks catalogs list` - List catalogs.

Gets an array of catalogs in the metastore.
If the caller is the metastore admin, all catalogs will be retrieved.
Otherwise, only catalogs owned by the caller (or for which the caller has the **USE_CATALOG** privilege) will be retrieved.
There is no guarantee of a specific ordering of the elements in the array.

### `bricks catalogs update` - Update a catalog.

Updates the catalog that matches the supplied name.
The caller must be either the owner of the catalog, or a metastore admin (when changing the owner field of the catalog).

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - User-provided free-form text description.
 * `--name` - Name of catalog.
 * `--owner` - Username of current owner of catalog.

## `bricks cluster-policies` - Cluster policy limits the ability to configure clusters based on a set of rules.

Cluster policy limits the ability to configure clusters based on a set of rules. The policy
rules limit the attributes or attribute values available for cluster creation. Cluster
policies have ACLs that limit their use to specific users and groups.

Cluster policies let you limit users to create clusters with prescribed settings, simplify
the user interface and enable more users to create their own clusters (by fixing and hiding
some values), control cost by limiting per cluster maximum cost (by setting limits on
attributes whose values contribute to hourly price).

Cluster policy permissions limit which policies a user can select in the Policy drop-down
when the user creates a cluster:
- A user who has cluster create permission can select the Unrestricted policy and create
  fully-configurable clusters.
- A user who has both cluster create permission and access to cluster policies can select
  the Unrestricted policy and policies they have access to.
- A user that has access to only cluster policies, can select the policies they have access to.

If no policies have been created in the workspace, the Policy drop-down does not display.

Only admin users can create, edit, and delete policies.
Admin users also have access to all policies.

### `bricks cluster-policies create` - Create a new policy.

Creates a new policy with prescribed settings.

Flags:
 * `--definition` - Policy definition document expressed in Databricks Cluster Policy Definition Language.
 * `--description` - Additional human-readable description of the cluster policy.
 * `--max-clusters-per-user` - Max number of clusters per user that can be active using this policy.
 * `--policy-family-definition-overrides` - Policy definition JSON document expressed in Databricks Policy Definition Language.
 * `--policy-family-id` - ID of the policy family.

### `bricks cluster-policies delete` - Delete a cluster policy.

Delete a policy for a cluster. Clusters governed by this policy can still run, but cannot be edited.

### `bricks cluster-policies edit` - Update a cluster policy.

Update an existing policy for cluster. This operation may make some clusters governed by the previous policy invalid.

Flags:
 * `--definition` - Policy definition document expressed in Databricks Cluster Policy Definition Language.
 * `--description` - Additional human-readable description of the cluster policy.
 * `--max-clusters-per-user` - Max number of clusters per user that can be active using this policy.
 * `--policy-family-definition-overrides` - Policy definition JSON document expressed in Databricks Policy Definition Language.
 * `--policy-family-id` - ID of the policy family.

### `bricks cluster-policies get` - Get entity.

Get a cluster policy entity. Creation and editing is available to admins only.

### `bricks cluster-policies list` - Get a cluster policy.

Returns a list of policies accessible by the requesting user.

Flags:
 * `--sort-column` - The cluster policy attribute to sort by.
 * `--sort-order` - The order in which the policies get listed.

## `bricks clusters` - The Clusters API allows you to create, start, edit, list, terminate, and delete clusters.

Databricks maps cluster node instance types to compute units known as DBUs. See the instance
type pricing page for a list of the supported instance types and their corresponding DBUs.

A Databricks cluster is a set of computation resources and configurations on which you run
data engineering, data science, and data analytics workloads, such as production
ETL pipelines, streaming analytics, ad-hoc analytics, and machine learning.

You run these workloads as a set of commands in a notebook or as an automated job.
Databricks makes a distinction between all-purpose clusters and job clusters. You use
all-purpose clusters to analyze data collaboratively using interactive notebooks. You use
job clusters to run fast and robust automated jobs.

You can create an all-purpose cluster using the UI, CLI, or commands. You can manually
terminate and restart an all-purpose cluster. Multiple users can share such clusters to do
collaborative interactive analysis.

IMPORTANT: Databricks retains cluster configuration information for up to 200 all-purpose
clusters terminated in the last 30 days and up to 30 job clusters recently terminated by
the job scheduler. To keep an all-purpose cluster configuration even after it has been
terminated for more than 30 days, an administrator can pin a cluster to the cluster list.

### `bricks clusters change-owner` - Change cluster owner.

Change the owner of the cluster. You must be an admin to perform this operation.

### `bricks clusters create` - Create new cluster.

Creates a new Spark cluster. This method will acquire new instances from the cloud provider if necessary.
This method is asynchronous; the returned `cluster_id` can be used to poll the cluster status.
When this method returns, the cluster will be in a `PENDING` state.
The cluster will be usable once it enters a `RUNNING` state.

Note: Databricks may not be able to acquire some of the requested nodes, due to cloud provider limitations
(account limits, spot price, etc.) or transient network issues.

If Databricks acquires at least 85% of the requested on-demand nodes, cluster creation will succeed.
Otherwise the cluster will terminate with an informative error message.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--apply-policy-default-values` - Note: This field won't be true for webapp requests.
 * `--autotermination-minutes` - Automatically terminates the cluster after it is inactive for this time in minutes.
 * `--cluster-name` - Cluster name requested by the user.
 * `--cluster-source` - Determines whether the cluster was created by a user through the UI, created by the Databricks Jobs Scheduler, or through an API request.
 * `--driver-instance-pool-id` - The optional ID of the instance pool for the driver of the cluster belongs.
 * `--driver-node-type-id` - The node type of the Spark driver.
 * `--enable-elastic-disk` - Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.
 * `--enable-local-disk-encryption` - Whether to enable LUKS on cluster VMs' local disks.
 * `--instance-pool-id` - The optional ID of the instance pool to which the cluster belongs.
 * `--node-type-id` - This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.
 * `--num-workers` - Number of worker nodes that this cluster should have.
 * `--policy-id` - The ID of the cluster policy used to create the cluster if applicable.
 * `--runtime-engine` - Decides which runtime engine to be use, e.g.

### `bricks clusters delete` - Terminate cluster.

Terminates the Spark cluster with the specified ID. The cluster is removed asynchronously.
Once the termination has completed, the cluster will be in a `TERMINATED` state.
If the cluster is already in a `TERMINATING` or `TERMINATED` state, nothing will happen.

Flags:
 * `--no-wait` - do not wait to reach TERMINATED state.
 * `--timeout` - maximum amount of time to reach TERMINATED state.

### `bricks clusters edit` - Update cluster configuration.

Updates the configuration of a cluster to match the provided attributes and size.
A cluster can be updated if it is in a `RUNNING` or `TERMINATED` state.

If a cluster is updated while in a `RUNNING` state, it will be restarted so that the new attributes can take effect.

If a cluster is updated while in a `TERMINATED` state, it will remain `TERMINATED`.
The next time it is started using the `clusters/start` API, the new attributes will take effect.
Any attempt to update a cluster in any other state will be rejected with an `INVALID_STATE` error code.

Clusters created by the Databricks Jobs service cannot be edited.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--apply-policy-default-values` - Note: This field won't be true for webapp requests.
 * `--autotermination-minutes` - Automatically terminates the cluster after it is inactive for this time in minutes.
 * `--cluster-name` - Cluster name requested by the user.
 * `--cluster-source` - Determines whether the cluster was created by a user through the UI, created by the Databricks Jobs Scheduler, or through an API request.
 * `--driver-instance-pool-id` - The optional ID of the instance pool for the driver of the cluster belongs.
 * `--driver-node-type-id` - The node type of the Spark driver.
 * `--enable-elastic-disk` - Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.
 * `--enable-local-disk-encryption` - Whether to enable LUKS on cluster VMs' local disks.
 * `--instance-pool-id` - The optional ID of the instance pool to which the cluster belongs.
 * `--node-type-id` - This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.
 * `--num-workers` - Number of worker nodes that this cluster should have.
 * `--policy-id` - The ID of the cluster policy used to create the cluster if applicable.
 * `--runtime-engine` - Decides which runtime engine to be use, e.g.

### `bricks clusters events` - List cluster activity events.

Retrieves a list of events about the activity of a cluster.
command is paginated. If there are more events to read, the response includes all the nparameters necessary to request
the next page of events.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--end-time` - The end time in epoch milliseconds.
 * `--limit` - The maximum number of events to include in a page of events.
 * `--offset` - The offset in the result set.
 * `--order` - The order to list events in; either "ASC" or "DESC".
 * `--start-time` - The start time in epoch milliseconds.

### `bricks clusters get` - Get cluster info.

"Retrieves the information for a cluster given its identifier.
Clusters can be described while they are running, or up to 60 days after they are terminated.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.

### `bricks clusters list` - List all clusters.

Return information about all pinned clusters, active clusters, up to 200 of the most recently terminated all-purpose clusters in
the past 30 days, and up to 30 of the most recently terminated job clusters in the past 30 days.

For example, if there is 1 pinned cluster, 4 active clusters, 45 terminated all-purpose clusters in the past 30 days,
and 50 terminated job clusters in the past 30 days, then command returns the 1 pinned cluster, 4 active clusters,
all 45 terminated all-purpose clusters, and the 30 most recently terminated job clusters.

Flags:
 * `--can-use-client` - Filter clusters based on what type of client it can be used for.

### `bricks clusters list-node-types` - List node types.

Returns a list of supported Spark node types. These node types can be used to launch a cluster.

### `bricks clusters list-zones` - List availability zones.

Returns a list of availability zones where clusters can be created in (For example, us-west-2a).
These zones can be used to launch a cluster.

### `bricks clusters permanent-delete` - Permanently delete cluster.

Permanently deletes a Spark cluster. This cluster is terminated and resources are asynchronously removed.

In addition, users will no longer see permanently deleted clusters in the cluster list, and API users can no longer
perform any action on permanently deleted clusters.

### `bricks clusters pin` - Pin cluster.

Pinning a cluster ensures that the cluster will always be returned by the ListClusters API.
Pinning a cluster that is already pinned will have no effect.
command can only be called by workspace admins.

### `bricks clusters resize` - Resize cluster.

Resizes a cluster to have a desired number of workers. This will fail unless the cluster is in a `RUNNING` state.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--num-workers` - Number of worker nodes that this cluster should have.

### `bricks clusters restart` - Restart cluster.

Restarts a Spark cluster with the supplied ID. If the cluster is not currently in a `RUNNING` state, nothing will happen.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.

### `bricks clusters spark-versions` - List available Spark versions.

Returns the list of available Spark versions. These versions can be used to launch a cluster.

### `bricks clusters start` - Start terminated cluster.

Starts a terminated Spark cluster with the supplied ID.
This works similar to `createCluster` except:

* The previous cluster id and attributes are preserved.
* The cluster starts with the last specified cluster size.
* If the previous cluster was an autoscaling cluster, the current cluster starts with the minimum number of nodes.
* If the cluster is not currently in a `TERMINATED` state, nothing will happen.
* Clusters launched to run a job cannot be started.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.

### `bricks clusters unpin` - Unpin cluster.

Unpinning a cluster will allow the cluster to eventually be removed from the ListClusters API.
Unpinning a cluster that is not pinned will have no effect.
command can only be called by workspace admins.

## `bricks account credentials` - These commands manage credential configurations for this workspace.

Databricks needs access to a cross-account service IAM role in your AWS account so that Databricks can deploy clusters
in the appropriate VPC for the new workspace. A credential configuration encapsulates this
role information, and its ID is used when creating a new workspace.

### `bricks account credentials create` - Create credential configuration.

Creates a Databricks credential configuration that represents cloud cross-account credentials for a specified account. Databricks uses this to set up network infrastructure properly to host Databricks clusters. For your AWS IAM role, you need to trust the External ID (the Databricks Account API account ID)  in the returned credential object, and configure the required access policy.

Save the response's `credentials_id` field, which is the ID for your new credential configuration object.

For information about how to create a new workspace with command, see [Create a new workspace using the Account API](http://docs.databricks.com/administration-guide/account-api/new-workspace.html)

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account credentials delete` - Delete credential configuration.

Deletes a Databricks credential configuration object for an account, both specified by ID. You cannot delete a credential that is associated with any workspace.

### `bricks account credentials get` - Get credential configuration.

Gets a Databricks credential configuration object for an account, both specified by ID.

### `bricks account credentials list` - Get all credential configurations.

Gets all Databricks credential configurations associated with an account specified by ID.

## `bricks current-user` - command allows retrieving information about currently authenticated user or service principal.

**NOTE** **this command may change**

command allows retrieving information about currently authenticated user or
service principal.

### `bricks current-user me` - Get current user info.

Get details about the current method caller's identity.

## `bricks account custom-app-integration` - manage custom oauth app integrations.

These commands enable administrators to manage custom oauth app integrations, which is required for
adding/using Custom OAuth App Integration like Tableau Cloud for Databricks in AWS cloud.

**Note:** You can only add/use the OAuth custom application integrations when OAuth enrollment
status is enabled.

### `bricks account custom-app-integration create` - Create Custom OAuth App Integration.

Create Custom OAuth App Integration.

You can retrieve the custom oauth app integration via :method:get.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--confidential` - indicates if an oauth client-secret should be generated.

### `bricks account custom-app-integration delete` - Delete Custom OAuth App Integration.

Delete an existing Custom OAuth App Integration.
You can retrieve the custom oauth app integration via :method:get.

### `bricks account custom-app-integration get` - Get OAuth Custom App Integration.

Gets the Custom OAuth App Integration for the given integration id.

### `bricks account custom-app-integration list` - Get custom oauth app integrations.

Get the list of custom oauth app integrations for the specified Databricks Account

### `bricks account custom-app-integration update` - Updates Custom OAuth App Integration.

Updates an existing custom OAuth App Integration.
You can retrieve the custom oauth app integration via :method:get.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks dashboards` - Databricks SQL Dashboards

Manage SQL Dashboards from CLI.

### `bricks dashboards create` - Create a dashboard object.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--dashboard-filters-enabled` - In the web application, query filters that share a name are coupled to a single selection box if this value is true.
 * `--is-draft` - Draft dashboards only appear in list views for their owners.
 * `--is-trashed` - Indicates whether the dashboard is trashed.
 * `--name` - The title of this dashboard that appears in list views and at the top of the dashboard page.
 * `--parent` - The identifier of the workspace folder containing the dashboard.

### `bricks dashboards delete` - Remove a dashboard.

Moves a dashboard to the trash. Trashed dashboards do not appear in list views or searches, and cannot be shared.

### `bricks dashboards get` - Retrieve a definition.

Returns a JSON representation of a dashboard object, including its visualization and query objects.

### `bricks dashboards list` - Get dashboard objects.

Fetch a paginated list of dashboard objects.

Flags:
 * `--order` - Name of dashboard attribute to order by.
 * `--page` - Page number to retrieve.
 * `--page-size` - Number of dashboards to return per page.
 * `--q` - Full text search term.

### `bricks dashboards restore` - Restore a dashboard.

A restored dashboard appears in list views and searches and can be shared.

## `bricks data-sources` - command is provided to assist you in making new query objects.

command is provided to assist you in making new query objects. When creating a query object,
you may optionally specify a `data_source_id` for the SQL warehouse against which it will run.
If you don't already know the `data_source_id` for your desired SQL warehouse, command will
help you find it.

command does not support searches. It returns the full list of SQL warehouses in your
workspace. We advise you to use any text editor, REST client, or `grep` to search the
response from command for the name of your SQL warehouse as it appears in Databricks SQL.

### `bricks data-sources list` - Get a list of SQL warehouses.

Retrieves a full list of SQL warehouses available in this workspace.
All fields that appear in command response are enumerated for clarity.
However, you need only a SQL warehouse's `id` to create new queries against it.

## `bricks account encryption-keys` - manage encryption key configurations.

These commands manage encryption key configurations for this workspace (optional). A key
configuration encapsulates the AWS KMS key information and some information about how
the key configuration can be used. There are two possible uses for key configurations:

* Managed services: A key configuration can be used to encrypt a workspace's notebook and
secret data in the control plane, as well as Databricks SQL queries and query history.
* Storage: A key configuration can be used to encrypt a workspace's DBFS and EBS data in
the data plane.

In both of these cases, the key configuration's ID is used when creating a new workspace.
This Preview feature is available if your account is on the E2 version of the platform.
Updating a running workspace with workspace storage encryption requires that the workspace
is on the E2 version of the platform. If you have an older workspace, it might not be on
the E2 version of the platform. If you are not sure, contact your Databricks representative.

### `bricks account encryption-keys create` - Create encryption key configuration.

Creates a customer-managed key configuration object for an account, specified by ID. This operation uploads a reference to a customer-managed key to Databricks. If the key is assigned as a workspace's customer-managed key for managed services, Databricks uses the key to encrypt the workspaces notebooks and secrets in the control plane, in addition to Databricks SQL queries and query history. If it is specified as a workspace's customer-managed key for workspace storage, the key encrypts the workspace's root S3 bucket (which contains the workspace's root DBFS and system data) and, optionally, cluster EBS volume data.

**Important**: Customer-managed keys are supported only for some deployment types, subscription types, and AWS regions.

This operation is available only if your account is on the E2 version of the platform or on a select custom plan that allows multiple workspaces per account.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account encryption-keys delete` - Delete encryption key configuration.

Deletes a customer-managed key configuration object for an account. You cannot delete a configuration that is associated with a running workspace.

### `bricks account encryption-keys get` - Get encryption key configuration.

Gets a customer-managed key configuration object for an account, specified by ID. This operation uploads a reference to a customer-managed key to Databricks. If assigned as a workspace's customer-managed key for managed services, Databricks uses the key to encrypt the workspaces notebooks and secrets in the control plane, in addition to Databricks SQL queries and query history. If it is specified as a workspace's customer-managed key for storage, the key encrypts the workspace's root S3 bucket (which contains the workspace's root DBFS and system data) and, optionally, cluster EBS volume data.

**Important**: Customer-managed keys are supported only for some deployment types, subscription types, and AWS regions.

This operation is available only if your account is on the E2 version of the platform.

### `bricks account encryption-keys list` - Get all encryption key configurations.

Gets all customer-managed key configuration objects for an account. If the key is specified as a workspace's managed services customer-managed key, Databricks uses the key to encrypt the workspace's notebooks and secrets in the control plane, in addition to Databricks SQL queries and query history. If the key is specified as a workspace's storage customer-managed key, the key is used to encrypt the workspace's root S3 bucket and optionally can encrypt cluster EBS volumes data in the data plane.

**Important**: Customer-managed keys are supported only for some deployment types, subscription types, and AWS regions.

This operation is available only if your account is on the E2 version of the platform.

## `bricks experiments` - Manage MLflow experiments

### `bricks experiments create-experiment` - Create experiment.

Creates an experiment with a name. Returns the ID of the newly created experiment.
Validates that another experiment with the same name does not already exist and fails
if another experiment with the same name already exists.

Throws `RESOURCE_ALREADY_EXISTS` if a experiment with the given name exists.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--artifact-location` - Location where all artifacts for the experiment are stored.

### `bricks experiments create-run` - Create a run.

Creates a new run within an experiment.
A run is usually a single execution of a machine learning or data ETL pipeline.
MLflow uses runs to track the `mlflowParam`, `mlflowMetric` and `mlflowRunTag` associated with a single execution.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--experiment-id` - ID of the associated experiment.
 * `--start-time` - Unix timestamp in milliseconds of when the run started.
 * `--user-id` - ID of the user executing the run.

### `bricks experiments delete-experiment` - Delete an experiment.

Marks an experiment and associated metadata, runs, metrics, params, and tags for deletion.
If the experiment uses FileStore, artifacts associated with experiment are also deleted.

### `bricks experiments delete-run` - Delete a run.

Marks a run for deletion.

### `bricks experiments delete-tag` - Delete a tag.

Deletes a tag on a run. Tags are run metadata that can be updated during a run and after a run completes.

### `bricks experiments get-by-name` - Get metadata.

Gets metadata for an experiment.

This endpoint will return deleted experiments, but prefers the active experiment if an active and deleted experiment
share the same name. If multiple deleted experiments share the same name, the API will return one of them.

Throws `RESOURCE_DOES_NOT_EXIST` if no experiment with the specified name exists.S

### `bricks experiments get-experiment` - Get an experiment.

Gets metadata for an experiment. This method works on deleted experiments.
Flags:



### `bricks experiments get-history` - Get history of a given metric within a run.

Gets a list of all values for the specified metric for a given run.

Flags:
 * `--max-results` - Maximum number of Metric records to return per paginated request.
 * `--run-id` - ID of the run from which to fetch metric values.
 * `--run-uuid` - [Deprecated, use run_id instead] ID of the run from which to fetch metric values.

### `bricks experiments get-run` - Get a run.

Gets the metadata, metrics, params, and tags for a run.
In the case where multiple metrics with the same key are logged for a run, return only the value
with the latest timestamp.

If there are multiple values with the latest timestamp, return the maximum of these values.

Flags:
 * `--run-uuid` - [Deprecated, use run_id instead] ID of the run to fetch.

### `bricks experiments list-artifacts` - Get all artifacts.

List artifacts for a run. Takes an optional `artifact_path` prefix. If it is specified, the response contains only artifacts with the specified prefix.",

Flags:
 * `--path` - Filter artifacts matching this path (a relative path from the root artifact directory).
 * `--run-id` - ID of the run whose artifacts to list.
 * `--run-uuid` - [Deprecated, use run_id instead] ID of the run whose artifacts to list.

### `bricks experiments list-experiments` - List experiments.

List experiments.

Gets a list of all experiments.
Flags:
 * `--max-results` - Maximum number of experiments desired.
 * `--view-type` - Qualifier for type of experiments to be returned.

### `bricks experiments log-batch` - Log a batch.

Logs a batch of metrics, params, and tags for a run.
If any data failed to be persisted, the server will respond with an error (non-200 status code).

In case of error (due to internal server error or an invalid request), partial data may be written.

You can write metrics, params, and tags in interleaving fashion, but within a given entity type are guaranteed to follow
the order specified in the request body.

The overwrite behavior for metrics,  params, and tags is as follows:

* Metrics: metric values are never overwritten.
  Logging a metric (key, value, timestamp) appends to the set of values for the metric with the provided key.

* Tags: tag values can be overwritten by successive writes to the same tag key.
  That is, if multiple tag values with the same key are provided in the same API request,
  the last-provided tag value is written. Logging the same tag (key, value) is permitted. Specifically, logging a tag is idempotent.

* Parameters: once written, param values cannot be changed (attempting to overwrite a param value will result in an error).
  However, logging the same param (key, value) is permitted. Specifically, logging a param is idempotent.

  Request Limits
  -------------------------------
  A single JSON-serialized API request may be up to 1 MB in size and contain:

 * No more than 1000 metrics,  params, and tags in total
 * Up to 1000 metrics - Up to 100  params
 * Up to 100 tags

 For example, a valid request might contain 900 metrics, 50 params, and 50 tags, but logging 900 metrics, 50 params,
 and 51 tags is invalid.

 The following limits also apply to metric, param, and tag keys and values:

 * Metric keyes, param keys, and tag keys can be up to 250 characters in length
 * Parameter and tag values can be up to 250 characters in length


Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--run-id` - ID of the run to log under.

### `bricks experiments log-metric` - Log a metric.

Logs a metric for a run. A metric is a key-value pair (string key, float value) with an associated timestamp.
Examples include the various metrics that represent ML model accuracy. A metric can be logged multiple times.

Flags:
 * `--run-id` - ID of the run under which to log the metric.
 * `--run-uuid` - [Deprecated, use run_id instead] ID of the run under which to log the metric.
 * `--step` - Step at which to log the metric.

### `bricks experiments log-model` - Log a model.

**NOTE:** Experimental: command may change or be removed in a future release without warning.

Flags:
 * `--model-json` - MLmodel file in json format.
 * `--run-id` - ID of the run to log under.

### `bricks experiments log-param` - Log a param.

Logs a param used for a run. A param is a key-value pair (string key, string value).
Examples include hyperparameters used for ML model training and constant dates and values used in an ETL pipeline.
A param can be logged only once for a run.

Flags:
 * `--run-id` - ID of the run under which to log the param.
 * `--run-uuid` - [Deprecated, use run_id instead] ID of the run under which to log the param.

### `bricks experiments restore-experiment` - Restores an experiment.

Restore an experiment marked for deletion. This also restores associated metadata, runs, metrics, params, and tags. If experiment uses FileStore, underlying artifacts associated with experiment are also restored.  Throws `RESOURCE_DOES_NOT_EXIST` if experiment was never created or was permanently deleted.",

### `bricks experiments restore-run` - Restore a run.

Restores a deleted run.

### `bricks experiments search-experiments` - Search experiments.

Searches for experiments that satisfy specified search criteria.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--filter` - String representing a SQL filter condition (e.g.
 * `--max-results` - Maximum number of experiments desired.
 * `--view-type` - Qualifier for type of experiments to be returned.

### `bricks experiments search-runs` - Search for runs.

Searches for runs that satisfy expressions.

Search expressions can use `mlflowMetric` and `mlflowParam` keys.",

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--filter` - A filter expression over params, metrics, and tags, that allows returning a subset of runs.
 * `--max-results` - Maximum number of runs desired.
 * `--run-view-type` - Whether to display only active, only deleted, or all runs.

### `bricks experiments set-experiment-tag` - Set a tag.

Sets a tag on an experiment. Experiment tags are metadata that can be updated.

### `bricks experiments set-tag` - Set a tag.

Sets a tag on a run. Tags are run metadata that can be updated during a run and after
a run completes.
Flags:
 * `--run-id` - ID of the run under which to log the tag.
 * `--run-uuid` - [Deprecated, use run_id instead] ID of the run under which to log the tag.

### `bricks experiments update-experiment` - Update an experiment.

Updates experiment metadata.
Flags:
 * `--new-name` - If provided, the experiment's name is changed to the new name.

### `bricks experiments update-run` - Update a run.

Updates run metadata.
Flags:
 * `--end-time` - Unix timestamp in milliseconds of when the run ended.
 * `--run-id` - ID of the run to update.
 * `--run-uuid` - [Deprecated, use run_id instead] ID of the run to update.
 * `--status` - Updated status of the run.

## `bricks external-locations` - manage cloud storage path with a storage credential that authorizes access to it.

An external location is an object that combines a cloud storage path with a storage
credential that authorizes access to the cloud storage path. Each external location is
subject to Unity Catalog access-control policies that control which users and groups can
access the credential. If a user does not have access to an external location in Unity
Catalog, the request fails and Unity Catalog does not attempt to authenticate to your cloud
tenant on the user’s behalf.

Databricks recommends using external locations rather than using storage credentials
directly.

To create external locations, you must be a metastore admin or a user with the
**CREATE_EXTERNAL_LOCATION** privilege.

### `bricks external-locations create` - Create an external location.

Creates a new external location entry in the metastore.
The caller must be a metastore admin or have the **CREATE_EXTERNAL_LOCATION** privilege on both the metastore and the associated storage credential.

Flags:
 * `--comment` - User-provided free-form text description.
 * `--read-only` - Indicates whether the external location is read-only.
 * `--skip-validation` - Skips validation of the storage credential associated with the external location.

### `bricks external-locations delete` - Delete an external location.

Deletes the specified external location from the metastore. The caller must be the owner of the external location.

Flags:
 * `--force` - Force deletion even if there are dependent external tables or mounts.

### `bricks external-locations get` - Get an external location.

Gets an external location from the metastore. The caller must be either a metastore admin, the owner of the external location, or a user that has some privilege on the external location.

### `bricks external-locations list` - List external locations.

Gets an array of external locations (__ExternalLocationInfo__ objects) from the metastore.
The caller must be a metastore admin, the owner of the external location, or a user that has some privilege on the external location.
There is no guarantee of a specific ordering of the elements in the array.

### `bricks external-locations update` - Update an external location.

Updates an external location in the metastore. The caller must be the owner of the external location, or be a metastore admin.
In the second case, the admin can only update the name of the external location.

Flags:
 * `--comment` - User-provided free-form text description.
 * `--credential-name` - Name of the storage credential used with this location.
 * `--force` - Force update even if changing url invalidates dependent external tables or mounts.
 * `--name` - Name of the external location.
 * `--owner` - The owner of the external location.
 * `--read-only` - Indicates whether the external location is read-only.
 * `--url` - Path URL of the external location.

## `bricks functions` - Functions implement User-Defined Functions (UDFs) in Unity Catalog.

The function implementation can be any SQL expression or Query, and it can be invoked wherever a table reference is allowed in a query.
In Unity Catalog, a function resides at the same level as a table, so it can be referenced with the form __catalog_name__.__schema_name__.__function_name__.

### `bricks functions create` - Create a function.

Creates a new function

The user must have the following permissions in order for the function to be created:
- **USE_CATALOG** on the function's parent catalog
- **USE_SCHEMA** and **CREATE_FUNCTION** on the function's parent schema

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
* `--comment` - User-provided free-form text description.
 * `--external-language` - External function language.
 * `--external-name` - External function name.
 * `--sql-path` - List of schemes whose objects can be referenced without qualification.

### `bricks functions delete` - Delete a function.

Deletes the function that matches the supplied name.
For the deletion to succeed, the user must satisfy one of the following conditions:
- Is the owner of the function's parent catalog
- Is the owner of the function's parent schema and have the **USE_CATALOG** privilege on its parent catalog
- Is the owner of the function itself and have both the **USE_CATALOG** privilege on its parent catalog and the **USE_SCHEMA** privilege on its parent schema

Flags:
 * `--force` - Force deletion even if the function is notempty.

### `bricks functions get` - Get a function.

Gets a function from within a parent catalog and schema.
For the fetch to succeed, the user must satisfy one of the following requirements:
- Is a metastore admin
- Is an owner of the function's parent catalog
- Have the **USE_CATALOG** privilege on the function's parent catalog and be the owner of the function
- Have the **USE_CATALOG** privilege on the function's parent catalog, the **USE_SCHEMA** privilege on the function's parent schema, and the **EXECUTE** privilege on the function itself

### `bricks functions list` - List functions.

List functions within the specified parent catalog and schema.
If the user is a metastore admin, all functions are returned in the output list.
Otherwise, the user must have the **USE_CATALOG** privilege on the catalog and the **USE_SCHEMA** privilege on the schema, and the output list contains only functions for which either the user has the **EXECUTE** privilege or the user is the owner.
There is no guarantee of a specific ordering of the elements in the array.

### `bricks functions update` - Update a function.

Updates the function that matches the supplied name.
Only the owner of the function can be updated. If the user is not a metastore admin, the user must be a member of the group that is the new function owner.
- Is a metastore admin
- Is the owner of the function's parent catalog
- Is the owner of the function's parent schema and has the **USE_CATALOG** privilege on its parent catalog
- Is the owner of the function itself and has the **USE_CATALOG** privilege on its parent catalog as well as the **USE_SCHEMA** privilege on the function's parent schema.

Flags:
 * `--owner` - Username of current owner of function.

## `bricks git-credentials` - Registers personal access token for Databricks to do operations on behalf of the user.

See [more info](https://docs.databricks.com/repos/get-access-tokens-from-git-provider.html).

### `bricks git-credentials create` - Create a credential entry.

Creates a Git credential entry for the user. Only one Git credential per user is
supported, so any attempts to create credentials if an entry already exists will
fail. Use the PATCH endpoint to update existing credentials, or the DELETE endpoint to
delete existing credentials.

Flags:
 * `--git-username` - Git username.
 * `--personal-access-token` - The personal access token used to authenticate to the corresponding Git provider.

### `bricks git-credentials delete` - Delete a credential.

Deletes the specified Git credential.

### `bricks git-credentials get` - Get a credential entry.

Gets the Git credential with the specified credential ID.

### `bricks git-credentials list` - Get Git credentials.

Lists the calling user's Git credentials. One credential per user is supported.

### `bricks git-credentials update` - Update a credential.

Updates the specified Git credential.
Flags:
 * `--git-provider` - Git provider.
 * `--git-username` - Git username.
 * `--personal-access-token` - The personal access token used to authenticate to the corresponding Git provider.

## `bricks global-init-scripts` - configure global initialization scripts for the workspace.

The Global Init Scripts API enables Workspace administrators to configure global
initialization scripts for their workspace. These scripts run on every node in every cluster
in the workspace.

**Important:** Existing clusters must be restarted to pick up any changes made to global
init scripts.
Global init scripts are run in order. If the init script returns with a bad exit code,
the Apache Spark container fails to launch and init scripts with later position are skipped.
If enough containers fail, the entire cluster fails with a `GLOBAL_INIT_SCRIPT_FAILURE`
error code.

### `bricks global-init-scripts create` - Create init script.

Creates a new global init script in this workspace.
Flags:
 * `--enabled` - Specifies whether the script is enabled.
 * `--position` - The position of a global init script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.

### `bricks global-init-scripts delete` - Delete init script.

Deletes a global init script.

### `bricks global-init-scripts get` - Get an init script.

Gets all the details of a script, including its Base64-encoded contents.

### `bricks global-init-scripts list` - Get init scripts.

Get a list of all global init scripts for this workspace. This returns all properties for each script but **not** the script contents.
To retrieve the contents of a script, use the [get a global init script](#operation/get-script) operation.

### `bricks global-init-scripts update` - Update init script.

Updates a global init script, specifying only the fields to change. All fields are optional.
Unspecified fields retain their current value.

Flags:
 * `--enabled` - Specifies whether the script is enabled.
 * `--position` - The position of a script, where 0 represents the first script to run, 1 is the second script to run, in ascending order.

## `bricks grants` - Manage data access in Unity Catalog.

In Unity Catalog, data is secure by default. Initially, users have no access to data in
a metastore. Access can be granted by either a metastore admin, the owner of an object, or
the owner of the catalog or schema that contains the object. Securable objects in Unity
Catalog are hierarchical and privileges are inherited downward.

Securable objects in Unity Catalog are hierarchical and privileges are inherited downward.
This means that granting a privilege on the catalog automatically grants the privilege to
all current and future objects within the catalog. Similarly, privileges granted on a schema
are inherited by all current and future objects within that schema.

### `bricks grants get` - Get permissions.

Gets the permissions for a securable.

Flags:
 * `--principal` - If provided, only the permissions for the specified principal (user or group) are returned.

### `bricks grants get-effective` - Get effective permissions.

Gets the effective permissions for a securable.
Flags:
 * `--principal` - If provided, only the effective permissions for the specified principal (user or group) are returned.

### `bricks grants update` - Update permissions.

Updates the permissions for a securable.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks groups` - Groups for identity management.

Groups simplify identity management, making it easier to assign access to Databricks Workspace, data,
and other securable objects.

It is best practice to assign access to workspaces and access-control policies in
Unity Catalog to groups, instead of to users individually. All Databricks Workspace identities can be
assigned as members of groups, and members inherit permissions that are assigned to their
group.

### `bricks groups create` - Create a new group.

Creates a group in the Databricks Workspace with a unique name, using the supplied group details.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--display-name` - String that represents a human-readable group name.
 * `--external-id` -
 * `--id` - Databricks group ID.

### `bricks groups delete` - Delete a group.

Deletes a group from the Databricks Workspace.

### `bricks groups get` - Get group details.

Gets the information for a specific group in the Databricks Workspace.

### `bricks groups list` - List group details.

Gets all details of the groups associated with the Databricks Workspace.

Flags:
 * `--attributes` - Comma-separated list of attributes to return in response.
 * `--count` - Desired number of results per page.
 * `--excluded-attributes` - Comma-separated list of attributes to exclude in response.
 * `--filter` - Query by which the results have to be filtered.
 * `--sort-by` - Attribute to sort the results.
 * `--sort-order` - The order to sort the results.
 * `--start-index` - Specifies the index of the first result.

### `bricks groups patch` - Update group details.

Partially updates the details of a group.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks groups update` - Replace a group.

Updates the details of a group by replacing the entire group entity.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--display-name` - String that represents a human-readable group name.
 * `--external-id` -
 * `--id` - Databricks group ID.

## `bricks account groups` - Account-level group management

Groups simplify identity management, making it easier to assign access to Databricks Account, data,
and other securable objects.

It is best practice to assign access to workspaces and access-control policies in
Unity Catalog to groups, instead of to users individually. All Databricks Account identities can be
assigned as members of groups, and members inherit permissions that are assigned to their
group.

### `bricks account groups create` - Create a new group.

Creates a group in the Databricks Account with a unique name, using the supplied group details.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--display-name` - String that represents a human-readable group name.
 * `--external-id` -
 * `--id` - Databricks group ID.

### `bricks account groups delete` - Delete a group.

Deletes a group from the Databricks Account.

### `bricks account groups get` - Get group details.

Gets the information for a specific group in the Databricks Account.

### `bricks account groups list` - List group details.

Gets all details of the groups associated with the Databricks Account.

Flags:
 * `--attributes` - Comma-separated list of attributes to return in response.
 * `--count` - Desired number of results per page.
 * `--excluded-attributes` - Comma-separated list of attributes to exclude in response.
 * `--filter` - Query by which the results have to be filtered.
 * `--sort-by` - Attribute to sort the results.
 * `--sort-order` - The order to sort the results.
 * `--start-index` - Specifies the index of the first result.

### `bricks account groups patch` - Update group details.

Partially updates the details of a group.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account groups update` - Replace a group.

Updates the details of a group by replacing the entire group entity.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--display-name` - String that represents a human-readable group name.
 * `--external-id` -
 * `--id` - Databricks group ID.

## `bricks instance-pools` - manage ready-to-use cloud instances which reduces a cluster start and auto-scaling times.

Instance Pools API are used to create, edit, delete and list instance pools by using
ready-to-use cloud instances which reduces a cluster start and auto-scaling times.

Databricks pools reduce cluster start and auto-scaling times by maintaining a set of idle,
ready-to-use instances. When a cluster is attached to a pool, cluster nodes are created using
the pool’s idle instances. If the pool has no idle instances, the pool expands by allocating
a new instance from the instance provider in order to accommodate the cluster’s request.
When a cluster releases an instance, it returns to the pool and is free for another cluster
to use. Only clusters attached to a pool can use that pool’s idle instances.

You can specify a different pool for the driver node and worker nodes, or use the same pool
for both.

Databricks does not charge DBUs while instances are idle in the pool. Instance provider
billing does apply. See pricing.

### `bricks instance-pools create` - Create a new instance pool.


Creates a new instance pool using idle and ready-to-use cloud instances.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--enable-elastic-disk` - Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.
 * `--idle-instance-autotermination-minutes` - Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.
 * `--max-capacity` - Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.
 * `--min-idle-instances` - Minimum number of idle instances to keep in the instance pool.

### `bricks instance-pools delete` - Delete an instance pool.

Deletes the instance pool permanently. The idle instances in the pool are terminated asynchronously.

### `bricks instance-pools edit` - Edit an existing instance pool.

Modifies the configuration of an existing instance pool.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--enable-elastic-disk` - Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.
 * `--idle-instance-autotermination-minutes` - Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.
 * `--max-capacity` - Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.
 * `--min-idle-instances` - Minimum number of idle instances to keep in the instance pool.

### `bricks instance-pools get` - Get instance pool information.

Retrieve the information for an instance pool based on its identifier.

### `bricks instance-pools list` - List instance pool info.

Gets a list of instance pools with their statistics.

## `bricks instance-profiles` - Manage instance profiles that users can launch clusters with.

The Instance Profiles API allows admins to add, list, and remove instance profiles that users can launch
clusters with. Regular users can list the instance profiles available to them.
See [Secure access to S3 buckets](https://docs.databricks.com/administration-guide/cloud-configurations/aws/instance-profiles.html) using
instance profiles for more information.

### `bricks instance-profiles add` - Register an instance profile.

In the UI, you can select the instance profile when launching clusters. command is only available to admin users.

Flags:
 * `--iam-role-arn` - The AWS IAM role ARN of the role associated with the instance profile.
 * `--is-meta-instance-profile` - By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.
 * `--skip-validation` - By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.

### `bricks instance-profiles edit` - Edit an instance profile.

The only supported field to change is the optional IAM role ARN associated with
the instance profile. It is required to specify the IAM role ARN if both of
the following are true:

 * Your role name and instance profile name do not match. The name is the part
   after the last slash in each ARN.
 * You want to use the instance profile with [Databricks SQL Serverless](https://docs.databricks.com/sql/admin/serverless.html).

To understand where these fields are in the AWS console, see
[Enable serverless SQL warehouses](https://docs.databricks.com/sql/admin/serverless.html).

command is only available to admin users.

Flags:
 * `--iam-role-arn` - The AWS IAM role ARN of the role associated with the instance profile.
 * `--is-meta-instance-profile` - By default, Databricks validates that it has sufficient permissions to launch instances with the instance profile.

### `bricks instance-profiles list` - List available instance profiles.

List the instance profiles that the calling user can use to launch a cluster.

command is available to all users.

### `bricks instance-profiles remove` - Remove the instance profile.

Remove the instance profile with the provided ARN.
Existing clusters with this instance profile will continue to function.

command is only accessible to admin users.

## `bricks ip-access-lists` - enable admins to configure IP access lists.

IP Access List enables admins to configure IP access lists.

IP access lists affect web application access and commands access to this workspace only.
If the feature is disabled for a workspace, all access is allowed for this workspace.
There is support for allow lists (inclusion) and block lists (exclusion).

When a connection is attempted:
  1. **First, all block lists are checked.** If the connection IP address matches any block list, the connection is rejected.
  2. **If the connection was not rejected by block lists**, the IP address is compared with the allow lists.

If there is at least one allow list for the workspace, the connection is allowed only if the IP address matches an allow list.
If there are no allow lists for the workspace, all IP addresses are allowed.

For all allow lists and block lists combined, the workspace supports a maximum of 1000 IP/CIDR values, where one CIDR counts as a single value.

After changes to the IP access list feature, it can take a few minutes for changes to take effect.

### `bricks ip-access-lists create` - Create access list.

Creates an IP access list for this workspace.

A list can be an allow list or a block list.
See the top of this file for a description of how the server treats allow lists and block lists at runtime.

When creating or updating an IP access list:

  * For all allow lists and block lists combined, the API supports a maximum of 1000 IP/CIDR values,
  where one CIDR counts as a single value. Attempts to exceed that number return error 400 with `error_code` value `QUOTA_EXCEEDED`.
  * If the new list would block the calling user's current IP, error 400 is returned with `error_code` value `INVALID_STATE`.

It can take a few minutes for the changes to take effect.
**Note**: Your new IP access list has no effect until you enable the feature. See :method:workspaceconf/setStatus

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks ip-access-lists delete` - Delete access list.

Deletes an IP access list, specified by its list ID.

### `bricks ip-access-lists get` - Get access list.

Gets an IP access list, specified by its list ID.

### `bricks ip-access-lists list` - Get access lists.

Gets all IP access lists for the specified workspace.

### `bricks ip-access-lists replace` - Replace access list.

Replaces an IP access list, specified by its ID.

A list can include allow lists and block lists. See the top
of this file for a description of how the server treats allow lists and block lists at run time. When
replacing an IP access list:
 * For all allow lists and block lists combined, the API supports a maximum of 1000 IP/CIDR values,
   where one CIDR counts as a single value. Attempts to exceed that number return error 400 with `error_code`
   value `QUOTA_EXCEEDED`.
 * If the resulting list would block the calling user's current IP, error 400 is returned with `error_code`
   value `INVALID_STATE`.
It can take a few minutes for the changes to take effect. Note that your resulting IP access list has no
effect until you enable the feature. See :method:workspaceconf/setStatus.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--list-id` - Universally unique identifier (UUID) of the IP access list.

### `bricks ip-access-lists update` - Update access list.

Updates an existing IP access list, specified by its ID.

A list can include allow lists and block lists.
See the top of this file for a description of how the server treats allow lists and block lists at run time.

When updating an IP access list:

  * For all allow lists and block lists combined, the API supports a maximum of 1000 IP/CIDR values,
  where one CIDR counts as a single value. Attempts to exceed that number return error 400 with `error_code` value `QUOTA_EXCEEDED`.
  * If the updated list would block the calling user's current IP, error 400 is returned with `error_code` value `INVALID_STATE`.

It can take a few minutes for the changes to take effect. Note that your resulting IP access list has no effect until you enable
the feature. See :method:workspaceconf/setStatus.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--list-id` - Universally unique identifier (UUID) of the IP access list.

## `bricks account ip-access-lists` - The Accounts IP Access List API enables account admins to configure IP access lists for access to the account console.

The Accounts IP Access List API enables account admins to configure IP access lists for
access to the account console.

Account IP Access Lists affect web application access and commands access to the account
console and account APIs. If the feature is disabled for the account, all access is allowed
for this account. There is support for allow lists (inclusion) and block lists (exclusion).

When a connection is attempted:
  1. **First, all block lists are checked.** If the connection IP address matches any block
  list, the connection is rejected.
  2. **If the connection was not rejected by block lists**, the IP address is compared with
  the allow lists.

If there is at least one allow list for the account, the connection is allowed only if the
IP address matches an allow list. If there are no allow lists for the account, all IP
addresses are allowed.

For all allow lists and block lists combined, the account supports a maximum of 1000 IP/CIDR
values, where one CIDR counts as a single value.

After changes to the account-level IP access lists, it can take a few minutes for changes
to take effect.

### `bricks account ip-access-lists create` - Create access list.

Creates an IP access list for the account.

A list can be an allow list or a block list. See the top of this file for a description of
how the server treats allow lists and block lists at runtime.

When creating or updating an IP access list:

  * For all allow lists and block lists combined, the API supports a maximum of 1000
  IP/CIDR values, where one CIDR counts as a single value. Attempts to exceed that number
  return error 400 with `error_code` value `QUOTA_EXCEEDED`.
  * If the new list would block the calling user's current IP, error 400 is returned with
  `error_code` value `INVALID_STATE`.

It can take a few minutes for the changes to take effect.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account ip-access-lists delete` - Delete access list.

Deletes an IP access list, specified by its list ID.

### `bricks account ip-access-lists get` - Get IP access list.

Gets an IP access list, specified by its list ID.

### `bricks account ip-access-lists list` - Get access lists.

Gets all IP access lists for the specified account.

### `bricks account ip-access-lists replace` - Replace access list.

Replaces an IP access list, specified by its ID.

A list can include allow lists and block lists. See the top of this file for a description
of how the server treats allow lists and block lists at run time. When replacing an IP
access list:
 * For all allow lists and block lists combined, the API supports a maximum of 1000 IP/CIDR values,
   where one CIDR counts as a single value. Attempts to exceed that number return error 400 with `error_code`
   value `QUOTA_EXCEEDED`.
 * If the resulting list would block the calling user's current IP, error 400 is returned with `error_code`
   value `INVALID_STATE`.
It can take a few minutes for the changes to take effect.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--list-id` - Universally unique identifier (UUID) of the IP access list.

### `bricks account ip-access-lists update` - Update access list.

Updates an existing IP access list, specified by its ID.

A list can include allow lists and block lists. See the top of this file for a description
of how the server treats allow lists and block lists at run time.

When updating an IP access list:

  * For all allow lists and block lists combined, the API supports a maximum of 1000
  IP/CIDR values, where one CIDR counts as a single value. Attempts to exceed that number
  return error 400 with `error_code` value `QUOTA_EXCEEDED`.
  * If the updated list would block the calling user's current IP, error 400 is returned
  with `error_code` value `INVALID_STATE`.

It can take a few minutes for the changes to take effect.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--list-id` - Universally unique identifier (UUID) of the IP access list.

## `bricks jobs` - Manage Databricks Workflows.

You can use a Databricks job to run a data processing or data analysis task in a Databricks
cluster with scalable resources. Your job can consist of a single task or can be a large,
multi-task workflow with complex dependencies. Databricks manages the task orchestration,
cluster management, monitoring, and error reporting for all of your jobs. You can run your
jobs immediately or periodically through an easy-to-use scheduling system. You can implement
job tasks using notebooks, JARS, Delta Live Tables pipelines, or Python, Scala, Spark
submit, and Java applications.

You should never hard code secrets or store them in plain text. Use the :service:secrets to manage secrets in the
[Databricks CLI](https://docs.databricks.com/dev-tools/cli/index.html).
Use the [Secrets utility](https://docs.databricks.com/dev-tools/databricks-utils.html#dbutils-secrets) to reference secrets in notebooks and jobs.

### `bricks jobs cancel-all-runs` - Cancel all runs of a job.

Cancels all active runs of a job. The runs are canceled asynchronously, so it doesn't
prevent new runs from being started.

### `bricks jobs cancel-run` - Cancel a job run.

Cancels a job run. The run is canceled asynchronously, so it may still be running when
this request completes.

Flags:
 * `--no-wait` - do not wait to reach TERMINATED or SKIPPED state.
 * `--timeout` - maximum amount of time to reach TERMINATED or SKIPPED state.

### `bricks jobs create` - Create a new job.

Create a new job.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--format` - Used to tell what is the format of the job.
 * `--max-concurrent-runs` - An optional maximum allowed number of concurrent runs of the job.
 * `--name` - An optional name for the job.
 * `--timeout-seconds` - An optional timeout applied to each run of this job.

### `bricks jobs delete` - Delete a job.

Deletes a job.

### `bricks jobs delete-run` - Delete a job run.

Deletes a non-active run. Returns an error if the run is active.

### `bricks jobs export-run` - Export and retrieve a job run.

Export and retrieve the job run task.

Flags:
 * `--views-to-export` - Which views to export (CODE, DASHBOARDS, or ALL).

### `bricks jobs get` - Get a single job.

Retrieves the details for a single job.

### `bricks jobs get-run` - Get a single job run.

Retrieve the metadata of a run.

Flags:
 * `--no-wait` - do not wait to reach TERMINATED or SKIPPED state.
 * `--timeout` - maximum amount of time to reach TERMINATED or SKIPPED state.
 * `--include-history` - Whether to include the repair history in the response.

### `bricks jobs get-run-output` - Get the output for a single run.

Retrieve the output and metadata of a single task run. When a notebook task returns
a value through the `dbutils.notebook.exit()` call, you can use this endpoint to retrieve
that value. Databricks restricts command to returning the first 5 MB of the output.
To return a larger result, you can store job results in a cloud storage service.

This endpoint validates that the __run_id__ parameter is valid and returns an HTTP status
code 400 if the __run_id__ parameter is invalid. Runs are automatically removed after
60 days. If you to want to reference them beyond 60 days, you must save old run results
before they expire.

### `bricks jobs list` - List all jobs.

Retrieves a list of jobs.

Flags:
 * `--expand-tasks` - Whether to include task and cluster details in the response.
 * `--limit` - The number of jobs to return.
 * `--name` - A filter on the list based on the exact (case insensitive) job name.
 * `--offset` - The offset of the first job to return, relative to the most recently created job.

### `bricks jobs list-runs` - List runs for a job.

List runs in descending order by start time.

Flags:
 * `--active-only` - If active_only is `true`, only active runs are included in the results; otherwise, lists both active and completed runs.
 * `--completed-only` - If completed_only is `true`, only completed runs are included in the results; otherwise, lists both active and completed runs.
 * `--expand-tasks` - Whether to include task and cluster details in the response.
 * `--job-id` - The job for which to list runs.
 * `--limit` - The number of runs to return.
 * `--offset` - The offset of the first run to return, relative to the most recent run.
 * `--run-type` - The type of runs to return.
 * `--start-time-from` - Show runs that started _at or after_ this value.
 * `--start-time-to` - Show runs that started _at or before_ this value.

### `bricks jobs repair-run` - Repair a job run.

Re-run one or more tasks. Tasks are re-run as part of the original job run.
They use the current job and task settings, and can be viewed in the history for the
original job run.

Flags:
 * `--no-wait` - do not wait to reach TERMINATED or SKIPPED state.
 * `--timeout` - maximum amount of time to reach TERMINATED or SKIPPED state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--latest-repair-id` - The ID of the latest repair.
 * `--rerun-all-failed-tasks` - If true, repair all failed tasks.

### `bricks jobs reset` - Overwrites all settings for a job.

Overwrites all the settings for a specific job. Use the Update endpoint to update job settings partially.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks jobs run-now` - Trigger a new job run.

Run a job and return the `run_id` of the triggered run.

Flags:
 * `--no-wait` - do not wait to reach TERMINATED or SKIPPED state.
 * `--timeout` - maximum amount of time to reach TERMINATED or SKIPPED state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--idempotency-token` - An optional token to guarantee the idempotency of job run requests.

### `bricks jobs submit` - Create and trigger a one-time run.

Submit a one-time run. This endpoint allows you to submit a workload directly without
creating a job. Runs submitted using this endpoint don’t display in the UI. Use the
`jobs/runs/get` API to check the run state after the job is submitted.

Flags:
 * `--no-wait` - do not wait to reach TERMINATED or SKIPPED state.
 * `--timeout` - maximum amount of time to reach TERMINATED or SKIPPED state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--idempotency-token` - An optional token that can be used to guarantee the idempotency of job run requests.
 * `--run-name` - An optional name for the run.
 * `--timeout-seconds` - An optional timeout applied to each run of this job.

### `bricks jobs update` - Partially updates a job.

Add, update, or remove specific settings of an existing job. Use the ResetJob to overwrite all job settings.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks libraries` - Manage libraries on a cluster.

The Libraries API allows you to install and uninstall libraries and get the status of
libraries on a cluster.

To make third-party or custom code available to notebooks and jobs running on your clusters,
you can install a library. Libraries can be written in Python, Java, Scala, and R. You can
upload Java, Scala, and Python libraries and point to external packages in PyPI, Maven, and
CRAN repositories.

Cluster libraries can be used by all notebooks running on a cluster. You can install a cluster
library directly from a public repository such as PyPI or Maven, using a previously installed
workspace library, or using an init script.

When you install a library on a cluster, a notebook already attached to that cluster will not
immediately see the new library. You must first detach and then reattach the notebook to
the cluster.

When you uninstall a library from a cluster, the library is removed only when you restart
the cluster. Until you restart the cluster, the status of the uninstalled library appears
as Uninstall pending restart.

### `bricks libraries all-cluster-statuses` - Get all statuses.

Get the status of all libraries on all clusters. A status will be available for all libraries installed on this cluster
via the API or the libraries UI as well as libraries set to be installed on all clusters via the libraries UI.

### `bricks libraries cluster-status` - Get status.

Get the status of libraries on a cluster. A status will be available for all libraries installed on this cluster via the API
or the libraries UI as well as libraries set to be installed on all clusters via the libraries UI.
The order of returned libraries will be as follows.

1. Libraries set to be installed on this cluster will be returned first.
  Within this group, the final order will be order in which the libraries were added to the cluster.

2. Libraries set to be installed on all clusters are returned next.
  Within this group there is no order guarantee.

3. Libraries that were previously requested on this cluster or on all clusters, but now marked for removal.
  Within this group there is no order guarantee.

### `bricks libraries install` - Add a library.

Add libraries to be installed on a cluster.
The installation is asynchronous; it happens in the background after the completion of this request.

**Note**: The actual set of libraries to be installed on a cluster is the union of the libraries specified via this method and
the libraries set to be installed on all clusters via the libraries UI.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks libraries uninstall` - Uninstall libraries.

Set libraries to be uninstalled on a cluster. The libraries won't be uninstalled until the cluster is restarted.
Uninstalling libraries that are not installed on the cluster will have no impact but is not an error.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks account log-delivery` - These commands manage log delivery configurations for this account.

These commands manage log delivery configurations for this account. The two supported log types
for command are _billable usage logs_ and _audit logs_. This feature is in Public Preview.
This feature works with all account ID types.

Log delivery works with all account types. However, if your account is on the E2 version of
the platform or on a select custom plan that allows multiple workspaces per account, you can
optionally configure different storage destinations for each workspace. Log delivery status
is also provided to know the latest status of log delivery attempts.
The high-level flow of billable usage delivery:

1. **Create storage**: In AWS, [create a new AWS S3 bucket](https://docs.databricks.com/administration-guide/account-api/aws-storage.html)
with a specific bucket policy. Using Databricks APIs, call the Account API to create a [storage configuration object](#operation/create-storage-config)
that uses the bucket name.
2. **Create credentials**: In AWS, create the appropriate AWS IAM role. For full details,
including the required IAM role policies and trust relationship, see
[Billable usage log delivery](https://docs.databricks.com/administration-guide/account-settings/billable-usage-delivery.html).
Using Databricks APIs, call the Account API to create a [credential configuration object](#operation/create-credential-config)
that uses the IAM role's ARN.
3. **Create log delivery configuration**: Using Databricks APIs, call the Account API to
[create a log delivery configuration](#operation/create-log-delivery-config) that uses
the credential and storage configuration objects from previous steps. You can specify if
the logs should include all events of that log type in your account (_Account level_ delivery)
or only events for a specific set of workspaces (_workspace level_ delivery). Account level
log delivery applies to all current and future workspaces plus account level logs, while
workspace level log delivery solely delivers logs related to the specified workspaces.
You can create multiple types of delivery configurations per account.

For billable usage delivery:
* For more information about billable usage logs, see
[Billable usage log delivery](https://docs.databricks.com/administration-guide/account-settings/billable-usage-delivery.html).
For the CSV schema, see the [Usage page](https://docs.databricks.com/administration-guide/account-settings/usage.html).
* The delivery location is `<bucket-name>/<prefix>/billable-usage/csv/`, where `<prefix>` is
the name of the optional delivery path prefix you set up during log delivery configuration.
Files are named `workspaceId=<workspace-id>-usageMonth=<month>.csv`.
* All billable usage logs apply to specific workspaces (_workspace level_ logs). You can
aggregate usage for your entire account by creating an _account level_ delivery
configuration that delivers logs for all current and future workspaces in your account.
* The files are delivered daily by overwriting the month's CSV file for each workspace.

For audit log delivery:
* For more information about about audit log delivery, see
[Audit log delivery](https://docs.databricks.com/administration-guide/account-settings/audit-logs.html),
which includes information about the used JSON schema.
* The delivery location is `<bucket-name>/<delivery-path-prefix>/workspaceId=<workspaceId>/date=<yyyy-mm-dd>/auditlogs_<internal-id>.json`.
Files may get overwritten with the same content multiple times to achieve exactly-once delivery.
* If the audit log delivery configuration included specific workspace IDs, only
_workspace-level_ audit logs for those workspaces are delivered. If the log delivery
configuration applies to the entire account (_account level_ delivery configuration),
the audit log delivery includes workspace-level audit logs for all workspaces in the account
as well as account-level audit logs. See
[Audit log delivery](https://docs.databricks.com/administration-guide/account-settings/audit-logs.html) for details.
* Auditable events are typically available in logs within 15 minutes.

### `bricks account log-delivery create` - Create a new log delivery configuration.

Creates a new Databricks log delivery configuration to enable delivery of the specified type of logs to your storage location. This requires that you already created a [credential object](#operation/create-credential-config) (which encapsulates a cross-account service IAM role) and a [storage configuration object](#operation/create-storage-config) (which encapsulates an S3 bucket).

For full details, including the required IAM role policies and bucket policies, see [Deliver and access billable usage logs](https://docs.databricks.com/administration-guide/account-settings/billable-usage-delivery.html) or [Configure audit logging](https://docs.databricks.com/administration-guide/account-settings/audit-logs.html).

**Note**: There is a limit on the number of log delivery configurations available per account (each limit applies separately to each log type including billable usage and audit logs). You can create a maximum of two enabled account-level delivery configurations (configurations without a workspace filter) per type. Additionally, you can create two enabled workspace-level delivery configurations per workspace for each log type, which means that the same workspace ID can occur in the workspace filter for no more than two delivery configurations per log type.

You cannot delete a log delivery configuration, but you can disable it (see [Enable or disable log delivery configuration](#operation/patch-log-delivery-config-status)).

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account log-delivery get` - Get log delivery configuration.

Gets a Databricks log delivery configuration object for an account, both specified by ID.

### `bricks account log-delivery list` - Get all log delivery configurations.

Gets all Databricks log delivery configurations associated with an account specified by ID.

Flags:
 * `--credentials-id` - Filter by credential configuration ID.
 * `--status` - Filter by status `ENABLED` or `DISABLED`.
 * `--storage-configuration-id` - Filter by storage configuration ID.

### `bricks account log-delivery patch-status` - Enable or disable log delivery configuration.

Enables or disables a log delivery configuration. Deletion of delivery configurations is not supported, so disable log delivery configurations that are no longer needed. Note that you can't re-enable a delivery configuration if this would violate the delivery configuration limits described under [Create log delivery](#operation/create-log-delivery-config).

## `bricks account metastore-assignments` - These commands manage metastore assignments to a workspace.

These commands manage metastore assignments to a workspace.

### `bricks account metastore-assignments create` - Assigns a workspace to a metastore.

Creates an assignment to a metastore for a workspace

### `bricks account metastore-assignments delete` - Delete a metastore assignment.

Deletes a metastore assignment to a workspace, leaving the workspace with no metastore.

### `bricks account metastore-assignments get` - Gets the metastore assignment for a workspace.

Gets the metastore assignment, if any, for the workspace specified by ID. If the workspace
is assigned a metastore, the mappig will be returned. If no metastore is assigned to the
workspace, the assignment will not be found and a 404 returned.

### `bricks account metastore-assignments list` - Get all workspaces assigned to a metastore.

Gets a list of all Databricks workspace IDs that have been assigned to given metastore.

### `bricks account metastore-assignments update` - Updates a metastore assignment to a workspaces.

Updates an assignment to a metastore for a workspace. Currently, only the default catalog
may be updated

Flags:
 * `--default-catalog-name` - The name of the default catalog for the metastore.
 * `--metastore-id` - The unique ID of the metastore.

## `bricks metastores` - Manage metastores in Unity Catalog.

A metastore is the top-level container of objects in Unity Catalog. It stores data assets
(tables and views) and the permissions that govern access to them. Databricks account admins
can create metastores and assign them to Databricks workspaces to control which workloads
use each metastore. For a workspace to use Unity Catalog, it must have a Unity Catalog
metastore attached.

Each metastore is configured with a root storage location in a cloud storage account.
This storage location is used for metadata and managed tables data.

NOTE: This metastore is distinct from the metastore included in Databricks workspaces
created before Unity Catalog was released. If your workspace includes a legacy Hive
metastore, the data in that metastore is available in a catalog named hive_metastore.

### `bricks metastores assign` - Create an assignment.

Creates a new metastore assignment.
If an assignment for the same __workspace_id__ exists, it will be overwritten by the new __metastore_id__ and
__default_catalog_name__. The caller must be an account admin.

### `bricks metastores create` - Create a metastore.

Creates a new metastore based on a provided name and storage root path.

Flags:
 * `--region` - Cloud region which the metastore serves (e.g., `us-west-2`, `westus`).

### `bricks metastores current` - Get metastore assignment for workspace.

Gets the metastore assignment for the workspace being accessed.

### `bricks metastores delete` - Delete a metastore.

Deletes a metastore. The caller must be a metastore admin.

Flags:
 * `--force` - Force deletion even if the metastore is not empty.

### `bricks metastores get` - Get a metastore.

Gets a metastore that matches the supplied ID. The caller must be a metastore admin to retrieve this info.

### `bricks metastores list` - List metastores.

Gets an array of the available metastores (as __MetastoreInfo__ objects). The caller must be an admin to retrieve this info.
There is no guarantee of a specific ordering of the elements in the array.

### `bricks metastores maintenance` - Enables or disables auto maintenance on the metastore.

Enables or disables auto maintenance on the metastore.

### `bricks metastores summary` - Get a metastore summary.

Gets information about a metastore. This summary includes the storage credential, the cloud vendor, the cloud region, and the global metastore ID.

### `bricks metastores unassign` - Delete an assignment.

Deletes a metastore assignment. The caller must be an account administrator.

### `bricks metastores update` - Update a metastore.

Updates information for a specific metastore. The caller must be a metastore admin.

Flags:
 * `--delta-sharing-organization-name` - The organization name of a Delta Sharing entity, to be used in Databricks-to-Databricks Delta Sharing as the official name.
 * `--delta-sharing-recipient-token-lifetime-in-seconds` - The lifetime of delta sharing recipient token in seconds.
 * `--delta-sharing-scope` - The scope of Delta Sharing enabled for the metastore.
 * `--name` - The user-specified name of the metastore.
 * `--owner` - The owner of the metastore.
 * `--privilege-model-version` - Privilege model version of the metastore, of the form `major.minor` (e.g., `1.0`).
 * `--storage-root-credential-id` - UUID of storage credential to access the metastore storage_root.

### `bricks metastores update-assignment` - Update an assignment.

Updates a metastore assignment. This operation can be used to update __metastore_id__ or __default_catalog_name__
for a specified Workspace, if the Workspace is already assigned a metastore.
The caller must be an account admin to update __metastore_id__; otherwise, the caller can be a Workspace admin.

Flags:
 * `--default-catalog-name` - The name of the default catalog for the metastore.
 * `--metastore-id` - The unique ID of the metastore.

## `bricks account metastores` - These commands manage Unity Catalog metastores for an account.

These commands manage Unity Catalog metastores for an account. A metastore contains catalogs
that can be associated with workspaces

### `bricks account metastores create` - Create metastore.

Creates a Unity Catalog metastore.

Flags:
 * `--region` - Cloud region which the metastore serves (e.g., `us-west-2`, `westus`).

### `bricks account metastores delete` - Delete a metastore.

Deletes a Databricks Unity Catalog metastore for an account, both specified by ID.

### `bricks account metastores get` - Get a metastore.

Gets a Databricks Unity Catalog metastore from an account, both specified by ID.

### `bricks account metastores list` - Get all metastores associated with an account.

Gets all Unity Catalog metastores associated with an account specified by ID.

### `bricks account metastores update` - Update a metastore.

Updates an existing Unity Catalog metastore.

Flags:
 * `--delta-sharing-organization-name` - The organization name of a Delta Sharing entity, to be used in Databricks-to-Databricks Delta Sharing as the official name.
 * `--delta-sharing-recipient-token-lifetime-in-seconds` - The lifetime of delta sharing recipient token in seconds.
 * `--delta-sharing-scope` - The scope of Delta Sharing enabled for the metastore.
 * `--name` - The user-specified name of the metastore.
 * `--owner` - The owner of the metastore.
 * `--privilege-model-version` - Privilege model version of the metastore, of the form `major.minor` (e.g., `1.0`).
 * `--storage-root-credential-id` - UUID of storage credential to access the metastore storage_root.

## `bricks model-registry` - Expose commands for Model Registry.

### `bricks model-registry approve-transition-request` - Approve transition request.

Approves a model version stage transition request.

Flags:
 * `--comment` - User-provided comment on the action.

### `bricks model-registry create-comment` - Post a comment.

Posts a comment on a model version. A comment can be submitted either by a user or programmatically to display
relevant information about the model. For example, test results or deployment errors.

### `bricks model-registry create-model` - Create a model.

Creates a new registered model with the name specified in the request body.

Throws `RESOURCE_ALREADY_EXISTS` if a registered model with the given name exists.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--description` - Optional description for registered model.

### `bricks model-registry create-model-version` - Create a model version.

Creates a model version.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--description` - Optional description for model version.
 * `--run-id` - MLflow run ID for correlation, if `source` was generated by an experiment run in MLflow tracking server.
 * `--run-link` - MLflow run link - this is the exact link of the run that generated this model version, potentially hosted at another instance of MLflow.

### `bricks model-registry create-transition-request` - Make a transition request.

Creates a model version stage transition request.

Flags:
 * `--comment` - User-provided comment on the action.

### `bricks model-registry create-webhook` - Create a webhook.

**NOTE**: This endpoint is in Public Preview.

Creates a registry webhook.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--description` - User-specified description for the webhook.
 * `--model-name` - Name of the model whose events would trigger this webhook.
 * `--status` - This describes an enum.

### `bricks model-registry delete-comment` - Delete a comment.

Deletes a comment on a model version.

### `bricks model-registry delete-model` - Delete a model.

Deletes a registered model.

### `bricks model-registry delete-model-tag` - Delete a model tag.

Deletes the tag for a registered model.

### `bricks model-registry delete-model-version` - Delete a model version.

Deletes a model version.

### `bricks model-registry delete-model-version-tag` - Delete a model version tag.

Deletes a model version tag.

### `bricks model-registry delete-transition-request` - Delete a ransition request.

Cancels a model version stage transition request.

Flags:
 * `--comment` - User-provided comment on the action.

### `bricks model-registry delete-webhook` - Delete a webhook.

**NOTE:** This endpoint is in Public Preview.

Deletes a registry webhook.

Flags:
 * `--id` - Webhook ID required to delete a registry webhook.

### `bricks model-registry get-latest-versions` - Get the latest version.

Gets the latest version of a registered model.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks model-registry get-model` - Get model.

Get the details of a model. This is a Databricks Workspace version of the [MLflow endpoint](https://www.mlflow.org/docs/latest/rest-api.html#get-registeredmodel)
that also returns the model's Databricks Workspace ID and the permission level of the requesting user on the model.

### `bricks model-registry get-model-version` - Get a model version.

Get a model version.

### `bricks model-registry get-model-version-download-uri` - Get a model version URI.

Gets a URI to download the model version.

### `bricks model-registry list-models` - List models.

Lists all available registered models, up to the limit specified in __max_results__.

Flags:
 * `--max-results` - Maximum number of registered models desired.
 * `--page-token` - Pagination token to go to the next page based on a previous query.

### `bricks model-registry list-transition-requests` - List transition requests.

Gets a list of all open stage transition requests for the model version.

### `bricks model-registry list-webhooks` - List registry webhooks.

**NOTE:** This endpoint is in Public Preview.

Lists all registry webhooks.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--model-name` - If not specified, all webhooks associated with the specified events are listed, regardless of their associated model.
 * `--page-token` - Token indicating the page of artifact results to fetch.

### `bricks model-registry reject-transition-request` - Reject a transition request.

Rejects a model version stage transition request.

Flags:
 * `--comment` - User-provided comment on the action.

### `bricks model-registry rename-model` - Rename a model.

Renames a registered model.

Flags:
 * `--new-name` - If provided, updates the name for this `registered_model`.

### `bricks model-registry search-model-versions` - Searches model versions.

Searches for specific model versions based on the supplied __filter__.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--filter` - String filter condition, like "name='my-model-name'".
 * `--max-results` - Maximum number of models desired.

### `bricks model-registry search-models` - Search models.

Search for registered models based on the specified __filter__.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--filter` - String filter condition, like "name LIKE 'my-model-name'".
 * `--max-results` - Maximum number of models desired.

### `bricks model-registry set-model-tag` - Set a tag.

Sets a tag on a registered model.

### `bricks model-registry set-model-version-tag` - Set a version tag.

Sets a model version tag.

### `bricks model-registry test-registry-webhook` - Test a webhook.

**NOTE:** This endpoint is in Public Preview.

Tests a registry webhook.

Flags:
 * `--event` - If `event` is specified, the test trigger uses the specified event.

### `bricks model-registry transition-stage` - Transition a stage.

Transition a model version's stage. This is a Databricks Workspace version of the [MLflow endpoint](https://www.mlflow.org/docs/latest/rest-api.html#transition-modelversion-stage)
that also accepts a comment associated with the transition to be recorded.",

Flags:
 * `--comment` - User-provided comment on the action.

### `bricks model-registry update-comment` - Update a comment.

Post an edit to a comment on a model version.

### `bricks model-registry update-model` - Update model.

Updates a registered model.

Flags:
 * `--description` - If provided, updates the description for this `registered_model`.

### `bricks model-registry update-model-version` - Update model version.

Updates the model version.

Flags:
 * `--description` - If provided, updates the description for this `registered_model`.

### `bricks model-registry update-webhook` - Update a webhook.

**NOTE:** This endpoint is in Public Preview.

Updates a registry webhook.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--description` - User-specified description for the webhook.
 * `--status` - This describes an enum.

## `bricks account networks` - Manage network configurations.

These commands manage network configurations for customer-managed VPCs (optional). Its ID is used when creating a new workspace if you use customer-managed VPCs.

### `bricks account networks create` - Create network configuration.

Creates a Databricks network configuration that represents an VPC and its resources. The VPC will be used for new Databricks clusters. This requires a pre-existing VPC and subnets.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--vpc-id` - The ID of the VPC associated with this network.

### `bricks account networks delete` - Delete a network configuration.

Deletes a Databricks network configuration, which represents a cloud VPC and its resources. You cannot delete a network that is associated with a workspace.

This operation is available only if your account is on the E2 version of the platform.

### `bricks account networks get` - Get a network configuration.

Gets a Databricks network configuration, which represents a cloud VPC and its resources.

### `bricks account networks list` - Get all network configurations.

Gets a list of all Databricks network configurations for an account, specified by ID.

This operation is available only if your account is on the E2 version of the platform.

## `bricks account o-auth-enrollment` - These commands enable administrators to enroll OAuth for their accounts, which is required for adding/using any OAuth published/custom application integration.

These commands enable administrators to enroll OAuth for their accounts, which is required for adding/using any OAuth published/custom application integration.

**Note:** Your account must be on the E2 version to use These commands, this is because OAuth
is only supported on the E2 version.

### `bricks account o-auth-enrollment create` - Create OAuth Enrollment request.

Create an OAuth Enrollment request to enroll OAuth for this account and optionally enable
the OAuth integration for all the partner applications in the account.

The parter applications are:
  - Power BI
  - Tableau Desktop
  - Databricks CLI

The enrollment is executed asynchronously, so the API will return 204 immediately. The
actual enrollment take a few minutes, you can check the status via API :method:get.

Flags:
 * `--enable-all-published-apps` - If true, enable OAuth for all the published applications in the account.

### `bricks account o-auth-enrollment get` - Get OAuth enrollment status.

Gets the OAuth enrollment status for this Account.

You can only add/use the OAuth published/custom application integrations when OAuth enrollment
status is enabled.

## `bricks permissions` - Manage access for various users on different objects and endpoints.

Permissions API are used to create read, write, edit, update and manage access for various
users on different objects and endpoints.

### `bricks permissions get` - Get object permissions.

Gets the permission of an object. Objects can inherit permissions from their parent objects or root objects.

### `bricks permissions get-permission-levels` - Get permission levels.

Gets the permission levels that a user can have on an object.

### `bricks permissions set` - Set permissions.

Sets permissions on object. Objects can inherit permissions from their parent objects and root objects.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks permissions update` - Update permission.

Updates the permissions on an object.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks pipelines` - Manage Delta Live Tables from command-line.

The Delta Live Tables API allows you to create, edit, delete, start, and view details about
pipelines.

Delta Live Tables is a framework for building reliable, maintainable, and testable data
processing pipelines. You define the transformations to perform on your data, and Delta Live
Tables manages task orchestration, cluster management, monitoring, data quality, and error
handling.

Instead of defining your data pipelines using a series of separate Apache Spark tasks, Delta
Live Tables manages how your data is transformed based on a target schema you define for each
processing step. You can also enforce data quality with Delta Live Tables expectations.
Expectations allow you to define expected data quality and specify how to handle records that
fail those expectations.

### `bricks pipelines create` - Create a pipeline.

Creates a new data processing pipeline based on the requested configuration. If successful, this method returns
the ID of the new pipeline.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--allow-duplicate-names` - If false, deployment will fail if name conflicts with that of another pipeline.
 * `--catalog` - A catalog in Unity Catalog to publish data from this pipeline to.
 * `--channel` - DLT Release Channel that specifies which version to use.
 * `--continuous` - Whether the pipeline is continuous or triggered.
 * `--development` - Whether the pipeline is in Development mode.
 * `--dry-run` -
 * `--edition` - Pipeline product edition.
 * `--id` - Unique identifier for this pipeline.
 * `--name` - Friendly identifier for this pipeline.
 * `--photon` - Whether Photon is enabled for this pipeline.
 * `--storage` - DBFS root directory for storing checkpoints and tables.
 * `--target` - Target schema (database) to add tables in this pipeline to.

### `bricks pipelines delete` - Delete a pipeline.

Deletes a pipeline.

### `bricks pipelines get` - Get a pipeline.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.

### `bricks pipelines get-update` - Get a pipeline update.

Gets an update from an active pipeline.

### `bricks pipelines list-pipeline-events` - List pipeline events.

Retrieves events for a pipeline.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--filter` - Criteria to select a subset of results, expressed using a SQL-like syntax.
 * `--max-results` - Max number of entries to return in a single page.
 * `--page-token` - Page token returned by previous call.

### `bricks pipelines list-pipelines` - List pipelines.

Lists pipelines defined in the Delta Live Tables system.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--filter` - Select a subset of results based on the specified criteria.
 * `--max-results` - The maximum number of entries to return in a single page.
 * `--page-token` - Page token returned by previous call.

### `bricks pipelines list-updates` - List pipeline updates.

List updates for an active pipeline.

Flags:
 * `--max-results` - Max number of entries to return in a single page.
 * `--page-token` - Page token returned by previous call.
 * `--until-update-id` - If present, returns updates until and including this update_id.

### `bricks pipelines reset` - Reset a pipeline.

Resets a pipeline.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.

### `bricks pipelines start-update` - Queue a pipeline update.

Starts or queues a pipeline update.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--cause` -
 * `--full-refresh` - If true, this update will reset all tables before running.

### `bricks pipelines stop` - Stop a pipeline.

Stops a pipeline.

Flags:
 * `--no-wait` - do not wait to reach IDLE state.
 * `--timeout` - maximum amount of time to reach IDLE state.

### `bricks pipelines update` - Edit a pipeline.

Updates a pipeline with the supplied configuration.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--allow-duplicate-names` - If false, deployment will fail if name has changed and conflicts the name of another pipeline.
 * `--catalog` - A catalog in Unity Catalog to publish data from this pipeline to.
 * `--channel` - DLT Release Channel that specifies which version to use.
 * `--continuous` - Whether the pipeline is continuous or triggered.
 * `--development` - Whether the pipeline is in Development mode.
 * `--edition` - Pipeline product edition.
 * `--expected-last-modified` - If present, the last-modified time of the pipeline settings before the edit.
 * `--id` - Unique identifier for this pipeline.
 * `--name` - Friendly identifier for this pipeline.
 * `--photon` - Whether Photon is enabled for this pipeline.
 * `--pipeline-id` - Unique identifier for this pipeline.
 * `--storage` - DBFS root directory for storing checkpoints and tables.
 * `--target` - Target schema (database) to add tables in this pipeline to.

## `bricks policy-families` - View available policy families.

View available policy families. A policy family contains a policy definition providing best
practices for configuring clusters for a particular use case.

Databricks manages and provides policy families for several common cluster use cases. You
cannot create, edit, or delete policy families.

Policy families cannot be used directly to create clusters. Instead, you create cluster
policies using a policy family. Cluster policies created using a policy family inherit the
policy family's policy definition.

### `bricks policy-families get` - get cluster policy family.

Do it.

### `bricks policy-families list` - list policy families.

Flags:
 * `--max-results` - The max number of policy families to return.
 * `--page-token` - A token that can be used to get the next page of results.

## `bricks account private-access` - PrivateLink settings.

These commands manage private access settings for this account.

### `bricks account private-access create` - Create private access settings.

Creates a private access settings object, which specifies how your workspace is
accessed over [AWS PrivateLink](https://aws.amazon.com/privatelink). To use AWS
PrivateLink, a workspace must have a private access settings object referenced
by ID in the workspace's `private_access_settings_id` property.

You can share one private access settings with multiple workspaces in a single account. However,
private access settings are specific to AWS regions, so only workspaces in the same
AWS region can use a given private access settings object.

Before configuring PrivateLink, read the
[Databricks article about PrivateLink](https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html).

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--private-access-level` - The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object.
 * `--public-access-enabled` - Determines if the workspace can be accessed over public internet.

### `bricks account private-access delete` - Delete a private access settings object.

Deletes a private access settings object, which determines how your workspace is accessed over [AWS PrivateLink](https://aws.amazon.com/privatelink).

Before configuring PrivateLink, read the [Databricks article about PrivateLink](https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html).

### `bricks account private-access get` - Get a private access settings object.

Gets a private access settings object, which specifies how your workspace is accessed over [AWS PrivateLink](https://aws.amazon.com/privatelink).

Before configuring PrivateLink, read the [Databricks article about PrivateLink](https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html).

### `bricks account private-access list` - Get all private access settings objects.

Gets a list of all private access settings objects for an account, specified by ID.

### `bricks account private-access replace` - Replace private access settings.

Updates an existing private access settings object, which specifies how your workspace is
accessed over [AWS PrivateLink](https://aws.amazon.com/privatelink). To use AWS
PrivateLink, a workspace must have a private access settings object referenced by ID in
the workspace's `private_access_settings_id` property.

This operation completely overwrites your existing private access settings object attached to your workspaces.
All workspaces attached to the private access settings are affected by any change.
If `public_access_enabled`, `private_access_level`, or `allowed_vpc_endpoint_ids`
are updated, effects of these changes might take several minutes to propagate to the
workspace API.

You can share one private access settings object with multiple
workspaces in a single account. However, private access settings are specific to
AWS regions, so only workspaces in the same AWS region can use a given private access
settings object.

Before configuring PrivateLink, read the
[Databricks article about PrivateLink](https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html).

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--private-access-level` - The private access level controls which VPC endpoints can connect to the UI or API of any workspace that attaches this private access settings object.
 * `--public-access-enabled` - Determines if the workspace can be accessed over public internet.

## `bricks providers` - Delta Sharing Providers commands.

Databricks Providers commands

### `bricks providers create` - Create an auth provider.

Creates a new authentication provider minimally based on a name and authentication type.
The caller must be an admin on the metastore.

Flags:
 * `--comment` - Description about the provider.
 * `--recipient-profile-str` - This field is required when the __authentication_type__ is **TOKEN** or not provided.

### `bricks providers delete` - Delete a provider.

Deletes an authentication provider, if the caller is a metastore admin or is the owner of the provider.

### `bricks providers get` - Get a provider.

Gets a specific authentication provider. The caller must supply the name of the provider, and must either be a metastore admin or the owner of the provider.

### `bricks providers list` - List providers.

Gets an array of available authentication providers.
The caller must either be a metastore admin or the owner of the providers.
Providers not owned by the caller are not included in the response.
There is no guarantee of a specific ordering of the elements in the array.

Flags:
 * `--data-provider-global-metastore-id` - If not provided, all providers will be returned.

### `bricks providers list-shares` - List shares by Provider.

Gets an array of a specified provider's shares within the metastore where:

  * the caller is a metastore admin, or
  * the caller is the owner.

### `bricks providers update` - Update a provider.

Updates the information for an authentication provider, if the caller is a metastore admin or is the owner of the provider.
If the update changes the provider name, the caller must be both a metastore admin and the owner of the provider.

Flags:
 * `--comment` - Description about the provider.
 * `--name` - The name of the Provider.
 * `--owner` - Username of Provider owner.
 * `--recipient-profile-str` - This field is required when the __authentication_type__ is **TOKEN** or not provided.

## `bricks account published-app-integration` - manage published OAuth app integrations like Tableau Cloud for Databricks in AWS cloud.

These commands enable administrators to manage published oauth app integrations, which is required for
adding/using Published OAuth App Integration like Tableau Cloud for Databricks in AWS cloud.

**Note:** You can only add/use the OAuth published application integrations when OAuth enrollment
status is enabled.

### `bricks account published-app-integration create` - Create Published OAuth App Integration.

Create Published OAuth App Integration.

You can retrieve the published oauth app integration via :method:get.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--app-id` - app_id of the oauth published app integration.

### `bricks account published-app-integration delete` - Delete Published OAuth App Integration.

Delete an existing Published OAuth App Integration.
You can retrieve the published oauth app integration via :method:get.

### `bricks account published-app-integration get` - Get OAuth Published App Integration.

Gets the Published OAuth App Integration for the given integration id.

### `bricks account published-app-integration list` - Get published oauth app integrations.

Get the list of published oauth app integrations for the specified Databricks Account

### `bricks account published-app-integration update` - Updates Published OAuth App Integration.

Updates an existing published OAuth App Integration. You can retrieve the published oauth app integration via :method:get.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks queries` - These endpoints are used for CRUD operations on query definitions.

These endpoints are used for CRUD operations on query definitions. Query definitions include
the target SQL warehouse, query text, name, description, tags, parameters, and visualizations.

### `bricks queries create` - Create a new query definition.

Creates a new query definition. Queries created with this endpoint belong to the authenticated user making the request.

The `data_source_id` field specifies the ID of the SQL warehouse to run this query against. You can use the Data Sources API to see a complete list of available SQL warehouses. Or you can copy the `data_source_id` from an existing query.

**Note**: You cannot add a visualization until you create the query.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--data-source-id` - The ID of the data source / SQL warehouse where this query will run.
 * `--description` - General description that can convey additional information about this query such as usage notes.
 * `--name` - The name or title of this query to display in list views.
 * `--parent` - The identifier of the workspace folder containing the query.
 * `--query` - The text of the query.

### `bricks queries delete` - Delete a query.

Moves a query to the trash.
Trashed queries immediately disappear from searches and list views, and they cannot be used for alerts.
The trash is deleted after 30 days.

### `bricks queries get` - Get a query definition.

Retrieve a query object definition along with contextual permissions information about the currently authenticated user.

### `bricks queries list` - Get a list of queries.

Gets a list of queries. Optionally, this list can be filtered by a search term.

Flags:
 * `--order` - Name of query attribute to order by.
 * `--page` - Page number to retrieve.
 * `--page-size` - Number of queries to return per page.
 * `--q` - Full text search term.

### `bricks queries restore` - Restore a query.

Restore a query that has been moved to the trash.
A restored query appears in list views and searches. You can use restored queries for alerts.

### `bricks queries update` - Change a query definition.

Modify this query definition.

**Note**: You cannot undo this operation.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--data-source-id` - The ID of the data source / SQL warehouse where this query will run.
 * `--description` - General description that can convey additional information about this query such as usage notes.
 * `--name` - The name or title of this query to display in list views.
 * `--query` - The text of the query.

## `bricks query-history` - Access the history of queries through SQL warehouses.

Access the history of queries through SQL warehouses.

### `bricks query-history list` - List Queries.

List the history of queries through SQL warehouses. You can filter by user ID, warehouse ID, status, and time range.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--include-metrics` - Whether to include metrics about query.
 * `--max-results` - Limit the number of results returned in one page.
 * `--page-token` - A token that can be used to get the next page of results.

## `bricks recipient-activation` - Delta Sharing recipient activation commands.

Databricks Recipient Activation commands

### `bricks recipient-activation get-activation-url-info` - Get a share activation URL.

Gets an activation URL for a share.

### `bricks recipient-activation retrieve-token` - Get an access token.

Retrieve access token with an activation url. This is a public API without any authentication.

## `bricks recipients` - Delta Sharing recipients.

Databricks Recipients commands

### `bricks recipients create` - Create a share recipient.

Creates a new recipient with the delta sharing authentication type in the metastore.
The caller must be a metastore admin or has the **CREATE_RECIPIENT** privilege on the metastore.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - Description about the recipient.
 * `--owner` - Username of the recipient owner.
 * `--sharing-code` - The one-time sharing code provided by the data recipient.

### `bricks recipients delete` - Delete a share recipient.

Deletes the specified recipient from the metastore. The caller must be the owner of the recipient.

### `bricks recipients get` - Get a share recipient.

Gets a share recipient from the metastore if:

  * the caller is the owner of the share recipient, or:
  * is a metastore admin

### `bricks recipients list` - List share recipients.

Gets an array of all share recipients within the current metastore where:

  * the caller is a metastore admin, or
  * the caller is the owner.

There is no guarantee of a specific ordering of the elements in the array.

Flags:
 * `--data-recipient-global-metastore-id` - If not provided, all recipients will be returned.

### `bricks recipients rotate-token` - Rotate a token.

Refreshes the specified recipient's delta sharing authentication token with the provided token info.
The caller must be the owner of the recipient.

### `bricks recipients share-permissions` - Get recipient share permissions.

Gets the share permissions for the specified Recipient. The caller must be a metastore admin or the owner of the Recipient.

### `bricks recipients update` - Update a share recipient.

Updates an existing recipient in the metastore. The caller must be a metastore admin or the owner of the recipient.
If the recipient name will be updated, the user must be both a metastore admin and the owner of the recipient.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - Description about the recipient.
 * `--name` - Name of Recipient.
 * `--owner` - Username of the recipient owner.

## `bricks repos` - Manage their git repos.

The Repos API allows users to manage their git repos. Users can use the API to access all
repos that they have manage permissions on.

Databricks Repos is a visual Git client in Databricks. It supports common Git operations
such a cloning a repository, committing and pushing, pulling, branch management, and visual
comparison of diffs when committing.

Within Repos you can develop code in notebooks or other files and follow data science and
engineering code development best practices using Git for version control, collaboration,
and CI/CD.

### `bricks repos create` - Create a repo.

Creates a repo in the workspace and links it to the remote Git repo specified.
Note that repos created programmatically must be linked to a remote Git repo, unlike repos created in the browser.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--path` - Desired path for the repo in the workspace.

### `bricks repos delete` - Delete a repo.

Deletes the specified repo.

### `bricks repos get` - Get a repo.

Returns the repo with the given repo ID.

### `bricks repos list` - Get repos.

Returns repos that the calling user has Manage permissions on. Results are paginated with each page containing twenty repos.

Flags:
 * `--next-page-token` - Token used to get the next page of results.
 * `--path-prefix` - Filters repos that have paths starting with the given path prefix.

### `bricks repos update` - Update a repo.

Updates the repo to a different branch or tag, or updates the repo to the latest commit on the same branch.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--branch` - Branch that the local version of the repo is checked out to.
 * `--tag` - Tag that the local version of the repo is checked out to.

## `bricks schemas` - Manage schemas in Unity Catalog.

A schema (also called a database) is the second layer of Unity Catalog’s three-level
namespace. A schema organizes tables, views and functions. To access (or list) a table or view in
a schema, users must have the USE_SCHEMA data permission on the schema and its parent catalog,
and they must have the SELECT permission on the table or view.

### `bricks schemas create` - Create a schema.

Creates a new schema for catalog in the Metatastore. The caller must be a metastore admin, or have the **CREATE_SCHEMA** privilege in the parent catalog.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - User-provided free-form text description.
 * `--storage-root` - Storage root URL for managed tables within schema.

### `bricks schemas delete` - Delete a schema.

Deletes the specified schema from the parent catalog. The caller must be the owner of the schema or an owner of the parent catalog.

### `bricks schemas get` - Get a schema.

Gets the specified schema within the metastore. The caller must be a metastore admin, the owner of the schema, or a user that has the **USE_SCHEMA** privilege on the schema.

### `bricks schemas list` - List schemas.

Gets an array of schemas for a catalog in the metastore. If the caller is the metastore admin or the owner of the parent catalog, all schemas for the catalog will be retrieved.
Otherwise, only schemas owned by the caller (or for which the caller has the **USE_SCHEMA** privilege) will be retrieved.
There is no guarantee of a specific ordering of the elements in the array.

### `bricks schemas update` - Update a schema.

Updates a schema for a catalog. The caller must be the owner of the schema or a metastore admin.
If the caller is a metastore admin, only the __owner__ field can be changed in the update.
If the __name__ field must be updated, the caller must be a metastore admin or have the **CREATE_SCHEMA** privilege on the parent catalog.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - User-provided free-form text description.
 * `--name` - Name of schema, relative to parent catalog.
 * `--owner` - Username of current owner of schema.

## `bricks secrets` - manage secrets, secret scopes, and access permissions.

The Secrets API allows you to manage secrets, secret scopes, and access permissions.

Sometimes accessing data requires that you authenticate to external data sources through JDBC.
Instead of directly entering your credentials into a notebook, use Databricks secrets to store
your credentials and reference them in notebooks and jobs.

Administrators, secret creators, and users granted permission can read Databricks secrets.
While Databricks makes an effort to redact secret values that might be displayed in notebooks,
it is not possible to prevent such users from reading secrets.

### `bricks secrets create-scope` - Create a new secret scope.

The scope name must consist of alphanumeric characters, dashes, underscores, and periods,
and may not exceed 128 characters. The maximum number of scopes in a workspace is 100.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--initial-manage-principal` - The principal that is initially granted `MANAGE` permission to the created scope.
 * `--scope-backend-type` - The backend type the scope will be created with.

### `bricks secrets delete-acl` - Delete an ACL.

Deletes the given ACL on the given scope.

Users must have the `MANAGE` permission to invoke command.
Throws `RESOURCE_DOES_NOT_EXIST` if no such secret scope, principal, or ACL exists.
Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets delete-scope` - Delete a secret scope.

Deletes a secret scope.

Throws `RESOURCE_DOES_NOT_EXIST` if the scope does not exist. Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets delete-secret` - Delete a secret.

Deletes the secret stored in this secret scope. You must have `WRITE` or `MANAGE` permission on the secret scope.

Throws `RESOURCE_DOES_NOT_EXIST` if no such secret scope or secret exists.
Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets get-acl` - Get secret ACL details.

Gets the details about the given ACL, such as the group and permission.
Users must have the `MANAGE` permission to invoke command.

Throws `RESOURCE_DOES_NOT_EXIST` if no such secret scope exists.
Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets list-acls` - Lists ACLs.

List the ACLs for a given secret scope. Users must have the `MANAGE` permission to invoke command.

Throws `RESOURCE_DOES_NOT_EXIST` if no such secret scope exists.
Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets list-scopes` - List all scopes.

Lists all secret scopes available in the workspace.

Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets list-secrets` - List secret keys.

Lists the secret keys that are stored at this scope.
This is a metadata-only operation; secret data cannot be retrieved using command.
Users need the READ permission to make this call.

The lastUpdatedTimestamp returned is in milliseconds since epoch.
Throws `RESOURCE_DOES_NOT_EXIST` if no such secret scope exists.
Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets put-acl` - Create/update an ACL.

Creates or overwrites the Access Control List (ACL) associated with the given principal (user or group) on the
specified scope point.

In general, a user or group will use the most powerful permission available to them,
and permissions are ordered as follows:

* `MANAGE` - Allowed to change ACLs, and read and write to this secret scope.
* `WRITE` - Allowed to read and write to this secret scope.
* `READ` - Allowed to read this secret scope and list what secrets are available.

Note that in general, secret values can only be read from within a command on a cluster (for example, through a notebook).
There is no API to read the actual secret value material outside of a cluster.
However, the user's permission will be applied based on who is executing the command, and they must have at least READ permission.

Users must have the `MANAGE` permission to invoke command.

The principal is a user or group name corresponding to an existing Databricks principal to be granted or revoked access.

Throws `RESOURCE_DOES_NOT_EXIST` if no such secret scope exists.
Throws `RESOURCE_ALREADY_EXISTS` if a permission for the principal already exists.
Throws `INVALID_PARAMETER_VALUE` if the permission is invalid.
Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

### `bricks secrets put-secret` - Add a secret.

Inserts a secret under the provided scope with the given name.
If a secret already exists with the same name, this command overwrites the existing secret's value.
The server encrypts the secret using the secret scope's encryption settings before storing it.

You must have `WRITE` or `MANAGE` permission on the secret scope.
The secret key must consist of alphanumeric characters, dashes, underscores, and periods, and cannot exceed 128 characters.
The maximum allowed secret value size is 128 KB. The maximum number of secrets in a given scope is 1000.

The input fields "string_value" or "bytes_value" specify the type of the secret, which will determine the value returned when
the secret value is requested. Exactly one must be specified.

Throws `RESOURCE_DOES_NOT_EXIST` if no such secret scope exists.
Throws `RESOURCE_LIMIT_EXCEEDED` if maximum number of secrets in scope is exceeded.
Throws `INVALID_PARAMETER_VALUE` if the key name or value length is invalid.
Throws `PERMISSION_DENIED` if the user does not have permission to make command call.

Flags:
 * `--bytes-value` - If specified, value will be stored as bytes.
 * `--string-value` - If specified, note that the value will be stored in UTF-8 (MB4) form.

## `bricks service-principals` - Manage service principals.

Identities for use with jobs, automated tools, and systems such as scripts, apps, and
CI/CD platforms. Databricks recommends creating service principals to run production jobs
or modify production data. If all processes that act on production data run with service
principals, interactive users do not need any write, delete, or modify privileges in
production. This eliminates the risk of a user overwriting production data by accident.

### `bricks service-principals create` - Create a service principal.

Creates a new service principal in the Databricks Workspace.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--application-id` - UUID relating to the service principal.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks service principal ID.

### `bricks service-principals delete` - Delete a service principal.

Delete a single service principal in the Databricks Workspace.

### `bricks service-principals get` - Get service principal details.

Gets the details for a single service principal define in the Databricks Workspace.

### `bricks service-principals list` - List service principals.

Gets the set of service principals associated with a Databricks Workspace.

Flags:
 * `--attributes` - Comma-separated list of attributes to return in response.
 * `--count` - Desired number of results per page.
 * `--excluded-attributes` - Comma-separated list of attributes to exclude in response.
 * `--filter` - Query by which the results have to be filtered.
 * `--sort-by` - Attribute to sort the results.
 * `--sort-order` - The order to sort the results.
 * `--start-index` - Specifies the index of the first result.


### `bricks service-principals patch` - Update service principal details.

Partially updates the details of a single service principal in the Databricks Workspace.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks service-principals update` - Replace service principal.

Updates the details of a single service principal.

This action replaces the existing service principal with the same name.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--application-id` - UUID relating to the service principal.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks service principal ID.

## `bricks account service-principals` - Manage service principals on the account level.

Identities for use with jobs, automated tools, and systems such as scripts, apps, and
CI/CD platforms. Databricks recommends creating service principals to run production jobs
or modify production data. If all processes that act on production data run with service
principals, interactive users do not need any write, delete, or modify privileges in
production. This eliminates the risk of a user overwriting production data by accident.

### `bricks account service-principals create` - Create a service principal.

Creates a new service principal in the Databricks Account.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--application-id` - UUID relating to the service principal.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks service principal ID.

### `bricks account service-principals delete` - Delete a service principal.

Delete a single service principal in the Databricks Account.

### `bricks account service-principals get` - Get service principal details.

Gets the details for a single service principal define in the Databricks Account.

### `bricks account service-principals list` - List service principals.

Gets the set of service principals associated with a Databricks Account.

Flags:
 * `--attributes` - Comma-separated list of attributes to return in response.
 * `--count` - Desired number of results per page.
 * `--excluded-attributes` - Comma-separated list of attributes to exclude in response.
 * `--filter` - Query by which the results have to be filtered.
 * `--sort-by` - Attribute to sort the results.
 * `--sort-order` - The order to sort the results.
 * `--start-index` - Specifies the index of the first result.

### `bricks account service-principals patch` - Update service principal details.

Partially updates the details of a single service principal in the Databricks Account.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account service-principals update` - Replace service principal.

Updates the details of a single service principal.

This action replaces the existing service principal with the same name.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--application-id` - UUID relating to the service principal.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks service principal ID.

## `bricks serving-endpoints` - Manage model serving endpoints.

The Serving Endpoints API allows you to create, update, and delete model serving endpoints.

You can use a serving endpoint to serve models from the Databricks Model Registry. Endpoints expose
the underlying models as scalable commands endpoints using serverless compute. This means
the endpoints and associated compute resources are fully managed by Databricks and will not appear in
your cloud account. A serving endpoint can consist of one or more MLflow models from the Databricks
Model Registry, called served models. A serving endpoint can have at most ten served models. You can configure
traffic settings to define how requests should be routed to your served models behind an endpoint.
Additionally, you can configure the scale of resources that should be applied to each served model.

### `bricks serving-endpoints build-logs` - Retrieve the logs associated with building the model's environment for a given serving endpoint's served model.

Retrieve the logs associated with building the model's environment for a given serving endpoint's served model.

Retrieves the build logs associated with the provided served model.

### `bricks serving-endpoints create` - Create a new serving endpoint.

Flags:
 * `--no-wait` - do not wait to reach NOT_UPDATING state.
 * `--timeout` - maximum amount of time to reach NOT_UPDATING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks serving-endpoints delete` - Delete a serving endpoint.

Delete a serving endpoint.

### `bricks serving-endpoints export-metrics` - Retrieve the metrics corresponding to a serving endpoint for the current time in Prometheus or OpenMetrics exposition format.

Retrieve the metrics corresponding to a serving endpoint for the current time in Prometheus or OpenMetrics exposition format.

Retrieves the metrics associated with the provided serving endpoint in either Prometheus or OpenMetrics exposition format.

### `bricks serving-endpoints get` - Get a single serving endpoint.

Retrieves the details for a single serving endpoint.

### `bricks serving-endpoints list` - Retrieve all serving endpoints.

Retrieve all serving endpoints.

### `bricks serving-endpoints logs` - Retrieve the most recent log lines associated with a given serving endpoint's served model.

Retrieves the service logs associated with the provided served model.

### `bricks serving-endpoints query` - Query a serving endpoint with provided model input.

Query a serving endpoint with provided model input.

### `bricks serving-endpoints update-config` - Update a serving endpoint with a new config.

Update a serving endpoint with a new config.

Updates any combination of the serving endpoint's served models, the compute
configuration of those served models, and the endpoint's traffic config.
An endpoint that already has an update in progress can not be updated until
the current update completes or fails.

Flags:
 * `--no-wait` - do not wait to reach NOT_UPDATING state.
 * `--timeout` - maximum amount of time to reach NOT_UPDATING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks shares` - Databricks Shares commands.

Databricks Shares commands

### `bricks shares create` - Create a share.

Creates a new share for data objects. Data objects can be added after creation with **update**.
The caller must be a metastore admin or have the **CREATE_SHARE** privilege on the metastore.

Flags:
 * `--comment` - User-provided free-form text description.

### `bricks shares delete` - Delete a share.

Deletes a data object share from the metastore. The caller must be an owner of the share.

### `bricks shares get` - Get a share.

Gets a data object share from the metastore. The caller must be a metastore admin or the owner of the share.

Flags:
 * `--include-shared-data` - Query for data to include in the share.

### `bricks shares list` - List shares.

Gets an array of data object shares from the metastore. The caller must be a metastore admin or the owner of the share.
There is no guarantee of a specific ordering of the elements in the array.

### `bricks shares share-permissions` - Get permissions.

Gets the permissions for a data share from the metastore.
The caller must be a metastore admin or the owner of the share.

### `bricks shares update` - Update a share.

Updates the share with the changes and data objects in the request.
The caller must be the owner of the share or a metastore admin.

When the caller is a metastore admin, only the __owner__ field can be updated.

In the case that the share name is changed, **updateShare** requires that the caller is both the share owner and
a metastore admin.

For each table that is added through this method, the share owner must also have **SELECT** privilege on the table.
This privilege must be maintained indefinitely for recipients to be able to access the table.
Typically, you should use a group as the share owner.

Table removals through **update** do not require additional privileges.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - User-provided free-form text description.
 * `--name` - Name of the share.
 * `--owner` - Username of current owner of share.

### `bricks shares update-permissions` - Update permissions.

Updates the permissions for a data share in the metastore.
The caller must be a metastore admin or an owner of the share.

For new recipient grants, the user must also be the owner of the recipients.
recipient revocations do not require additional privileges.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks account storage` - Manage storage configurations for this workspace.

These commands manage storage configurations for this workspace. A root storage S3 bucket in
your account is required to store objects like cluster logs, notebook revisions, and job
results. You can also use the root storage S3 bucket for storage of non-production DBFS
data. A storage configuration encapsulates this bucket information, and its ID is used when
creating a new workspace.

### `bricks account storage create` - Create new storage configuration.

Creates new storage configuration for an account, specified by ID. Uploads a storage configuration object that represents the root AWS S3 bucket in your account. Databricks stores related workspace assets including DBFS, cluster logs, and job results. For the AWS S3 bucket, you need to configure the required bucket policy.

For information about how to create a new workspace with command, see [Create a new workspace using the Account API](http://docs.databricks.com/administration-guide/account-api/new-workspace.html)

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account storage delete` - Delete storage configuration.

Deletes a Databricks storage configuration. You cannot delete a storage configuration that is associated with any workspace.

### `bricks account storage get` - Get storage configuration.

Gets a Databricks storage configuration for an account, both specified by ID.

### `bricks account storage list` - Get all storage configurations.

Gets a list of all Databricks storage configurations for your account, specified by ID.

## `bricks storage-credentials` - Manage storage credentials for Unity Catalog.

A storage credential represents an authentication and authorization mechanism for accessing
data stored on your cloud tenant. Each storage credential is subject to
Unity Catalog access-control policies that control which users and groups can access
the credential. If a user does not have access to a storage credential in Unity Catalog,
the request fails and Unity Catalog does not attempt to authenticate to your cloud tenant
on the user’s behalf.

Databricks recommends using external locations rather than using storage credentials
directly.

To create storage credentials, you must be a Databricks account admin. The account admin
who creates the storage credential can delegate ownership to another user or group to
manage permissions on it.

### `bricks storage-credentials create` - Create a storage credential.

Creates a new storage credential. The request object is specific to the cloud:

  * **AwsIamRole** for AWS credentials
  * **AzureServicePrincipal** for Azure credentials
  * **GcpServiceAcountKey** for GCP credentials.

The caller must be a metastore admin and have the **CREATE_STORAGE_CREDENTIAL** privilege on the metastore.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - Comment associated with the credential.
 * `--read-only` - Whether the storage credential is only usable for read operations.
 * `--skip-validation` - Supplying true to this argument skips validation of the created credential.

### `bricks storage-credentials delete` - Delete a credential.

Deletes a storage credential from the metastore. The caller must be an owner of the storage credential.

Flags:
 * `--force` - Force deletion even if there are dependent external locations or external tables.

### `bricks storage-credentials get` - Get a credential.

Gets a storage credential from the metastore. The caller must be a metastore admin, the owner of the storage credential, or have some permission on the storage credential.

### `bricks storage-credentials list` - List credentials.

Gets an array of storage credentials (as __StorageCredentialInfo__ objects).
The array is limited to only those storage credentials the caller has permission to access.
If the caller is a metastore admin, all storage credentials will be retrieved.
There is no guarantee of a specific ordering of the elements in the array.

### `bricks storage-credentials update` - Update a credential.

Updates a storage credential on the metastore. The caller must be the owner of the storage credential or a metastore admin. If the caller is a metastore admin, only the __owner__ credential can be changed.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--comment` - Comment associated with the credential.
 * `--force` - Force update even if there are dependent external locations or external tables.
 * `--name` - The credential name.
 * `--owner` - Username of current owner of credential.
 * `--read-only` - Whether the storage credential is only usable for read operations.
 * `--skip-validation` - Supplying true to this argument skips validation of the updated credential.

### `bricks storage-credentials validate` - Validate a storage credential.

Validates a storage credential. At least one of __external_location_name__ and __url__ need to be provided. If only one of them is
provided, it will be used for validation. And if both are provided, the __url__ will be used for
validation, and __external_location_name__ will be ignored when checking overlapping urls.

Either the __storage_credential_name__ or the cloud-specific credential must be provided.

The caller must be a metastore admin or the storage credential owner or
have the **CREATE_EXTERNAL_LOCATION** privilege on the metastore and the storage credential.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--external-location-name` - The name of an existing external location to validate.
 * `--read-only` - Whether the storage credential is only usable for read operations.
 * `--url` - The external location url to validate.

## `bricks account storage-credentials` - These commands manage storage credentials for a particular metastore.

These commands manage storage credentials for a particular metastore.

### `bricks account storage-credentials create` - Create a storage credential.

Creates a new storage credential. The request object is specific to the cloud:

  * **AwsIamRole** for AWS credentials
  * **AzureServicePrincipal** for Azure credentials
  * **GcpServiceAcountKey** for GCP credentials.

The caller must be a metastore admin and have the **CREATE_STORAGE_CREDENTIAL** privilege on the metastore.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body.
 * `--comment` - Comment associated with the credential.
 * `--read-only` - Whether the storage credential is only usable for read operations.
 * `--skip-validation` - Supplying true to this argument skips validation of the created credential.

### `bricks account storage-credentials get` - Gets the named storage credential.

Gets a storage credential from the metastore. The caller must be a metastore admin, the owner of the storage credential, or have a level of privilege on the storage credential.

### `bricks account storage-credentials list` - Get all storage credentials assigned to a metastore.

Gets a list of all storage credentials that have been assigned to given metastore.

## `bricks table-constraints` - Primary key and foreign key constraints encode relationships between fields in tables.

Primary and foreign keys are informational only and are not enforced. Foreign keys must reference a primary key in another table.
This primary key is the parent constraint of the foreign key and the table this primary key is on is the parent table of the foreign key.
Similarly, the foreign key is the child constraint of its referenced primary key; the table of the foreign key is the child table of the primary key.

You can declare primary keys and foreign keys as part of the table specification during table creation.
You can also add or drop constraints on existing tables.

### `bricks table-constraints create` - Create a table constraint.


For the table constraint creation to succeed, the user must satisfy both of these conditions:
- the user must have the **USE_CATALOG** privilege on the table's parent catalog,
  the **USE_SCHEMA** privilege on the table's parent schema, and be the owner of the table.
- if the new constraint is a __ForeignKeyConstraint__,
  the user must have the **USE_CATALOG** privilege on the referenced parent table's catalog,
  the **USE_SCHEMA** privilege on the referenced parent table's schema,
  and be the owner of the referenced parent table.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks table-constraints delete` - Delete a table constraint.

Deletes a table constraint.

For the table constraint deletion to succeed, the user must satisfy both of these conditions:
- the user must have the **USE_CATALOG** privilege on the table's parent catalog,
  the **USE_SCHEMA** privilege on the table's parent schema, and be the owner of the table.
- if __cascade__ argument is **true**, the user must have the following permissions on all of the child tables:
  the **USE_CATALOG** privilege on the table's catalog,
  the **USE_SCHEMA** privilege on the table's schema,
  and be the owner of the table.

## `bricks tables` - A table resides in the third layer of Unity Catalog’s three-level namespace.

A table resides in the third layer of Unity Catalog’s three-level namespace. It contains
rows of data. To create a table, users must have CREATE_TABLE and USE_SCHEMA permissions on the schema,
and they must have the USE_CATALOG permission on its parent catalog. To query a table, users must
have the SELECT permission on the table, and they must have the USE_CATALOG permission on its
parent catalog and the USE_SCHEMA permission on its parent schema.

A table can be managed or external. From an API perspective, a __VIEW__ is a particular kind of table (rather than a managed or external table).

### `bricks tables delete` - Delete a table.

Deletes a table from the specified parent catalog and schema.
The caller must be the owner of the parent catalog, have the **USE_CATALOG** privilege on the parent catalog and be the owner of the parent schema,
or be the owner of the table and have the **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent schema.

### `bricks tables get` - Get a table.

Gets a table from the metastore for a specific catalog and schema.
The caller must be a metastore admin, be the owner of the table and have the **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent schema,
or be the owner of the table and have the **SELECT** privilege on it as well.

Flags:
 * `--include-delta-metadata` - Whether delta metadata should be included in the response.

### `bricks tables list` - List tables.

Gets an array of all tables for the current metastore under the parent catalog and schema.
The caller must be a metastore admin or an owner of (or have the **SELECT** privilege on) the table.
For the latter case, the caller must also be the owner or have the **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent schema.
There is no guarantee of a specific ordering of the elements in the array.

Flags:
 * `--include-delta-metadata` - Whether delta metadata should be included in the response.
 * `--max-results` - Maximum number of tables to return (page length).
 * `--page-token` - Opaque token to send for the next page of results (pagination).

### `bricks tables list-summaries` - List table summaries.

Gets an array of summaries for tables for a schema and catalog within the metastore. The table summaries returned are either:

* summaries for all tables (within the current metastore and parent catalog and schema), when the user is a metastore admin, or:
* summaries for all tables and schemas (within the current metastore and parent catalog)
  for which the user has ownership or the **SELECT** privilege on the table and ownership or **USE_SCHEMA** privilege on the schema,
  provided that the user also has ownership or the **USE_CATALOG** privilege on the parent catalog.

There is no guarantee of a specific ordering of the elements in the array.

Flags:
 * `--max-results` - Maximum number of tables to return (page length).
 * `--page-token` - Opaque token to send for the next page of results (pagination).
 * `--schema-name-pattern` - A sql LIKE pattern (% and _) for schema names.
 * `--table-name-pattern` - A sql LIKE pattern (% and _) for table names.

## `bricks token-management` - Enables administrators to get all tokens and delete tokens for other users.

Enables administrators to get all tokens and delete tokens for other users. Admins can
either get every token, get a specific token by ID, or get all tokens for a particular user.

### `bricks token-management create-obo-token` - Create on-behalf token.

Creates a token on behalf of a service principal.

Flags:
 * `--comment` - Comment that describes the purpose of the token.

### `bricks token-management delete` - Delete a token.

Deletes a token, specified by its ID.

### `bricks token-management get` - Get token info.

Gets information about a token, specified by its ID.

### `bricks token-management list` - List all tokens.

Lists all tokens associated with the specified workspace or user.

Flags:
 * `--created-by-id` - User ID of the user that created the token.
 * `--created-by-username` - Username of the user that created the token.

## `bricks tokens` - The Token API allows you to create, list, and revoke tokens that can be used to authenticate and access Databricks commandss.

The Token API allows you to create, list, and revoke tokens that can be used to authenticate and access Databricks commandss.

### `bricks tokens create` - Create a user token.

Creates and returns a token for a user. If this call is made through token authentication, it creates
a token with the same client ID as the authenticated token. If the user's token quota is exceeded, this call
returns an error **QUOTA_EXCEEDED**.

Flags:
 * `--comment` - Optional description to attach to the token.
 * `--lifetime-seconds` - The lifetime of the token, in seconds.

### `bricks tokens delete` - Revoke token.

Revokes an access token.

If a token with the specified ID is not valid, this call returns an error **RESOURCE_DOES_NOT_EXIST**.

### `bricks tokens list` - List tokens.

Lists all the valid tokens for a user-workspace pair.

## `bricks users` - Manage users on the workspace-level.

Databricks recommends using SCIM provisioning to sync users and groups automatically from
your identity provider to your Databricks Workspace. SCIM streamlines onboarding a new
employee or team by using your identity provider to create users and groups in Databricks Workspace
and give them the proper level of access. When a user leaves your organization or no longer
needs access to Databricks Workspace, admins can terminate the user in your identity provider and that
user’s account will also be removed from Databricks Workspace. This ensures a consistent offboarding
process and prevents unauthorized users from accessing sensitive data.

### `bricks users create` - Create a new user.

Creates a new user in the Databricks Workspace. This new user will also be added to the Databricks account.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks user ID.
 * `--user-name` - Email address of the Databricks user.

### `bricks users delete` - Delete a user.

Deletes a user. Deleting a user from a Databricks Workspace also removes objects associated with the user.

### `bricks users get` - Get user details.

Gets information for a specific user in Databricks Workspace.

### `bricks users list` - List users.

Gets details for all the users associated with a Databricks Workspace.

Flags:
 * `--attributes` - Comma-separated list of attributes to return in response.
 * `--count` - Desired number of results per page.
 * `--excluded-attributes` - Comma-separated list of attributes to exclude in response.
 * `--filter` - Query by which the results have to be filtered.
 * `--sort-by` - Attribute to sort the results.
 * `--sort-order` - The order to sort the results.
 * `--start-index` - Specifies the index of the first result.

### `bricks users patch` - Update user details.

Partially updates a user resource by applying the supplied operations on specific user attributes.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks users update` - Replace a user.

Replaces a user's information with the data supplied in request.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks user ID.
 * `--user-name` - Email address of the Databricks user.

## `bricks account users` - Manage users on the accou

Databricks recommends using SCIM provisioning to sync users and groups automatically from
your identity provider to your Databricks Account. SCIM streamlines onboarding a new
employee or team by using your identity provider to create users and groups in Databricks Account
and give them the proper level of access. When a user leaves your organization or no longer
needs access to Databricks Account, admins can terminate the user in your identity provider and that
user’s account will also be removed from Databricks Account. This ensures a consistent offboarding
process and prevents unauthorized users from accessing sensitive data.

### `bricks account users create` - Create a new user.

Creates a new user in the Databricks Account. This new user will also be added to the Databricks account.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks user ID.
 * `--user-name` - Email address of the Databricks user.

### `bricks account users delete` - Delete a user.

Deleting a user from a Databricks Account also removes objects associated with the user.

### `bricks account users get` - Get user details.

Gets information for a specific user in Databricks Account.

### `bricks account users list` - List users.

Gets details for all the users associated with a Databricks Account.

Flags:
 * `--attributes` - Comma-separated list of attributes to return in response.
 * `--count` - Desired number of results per page.
 * `--excluded-attributes` - Comma-separated list of attributes to exclude in response.
 * `--filter` - Query by which the results have to be filtered.
 * `--sort-by` - Attribute to sort the results.
 * `--sort-order` - The order to sort the results.
 * `--start-index` - Specifies the index of the first result.

### `bricks account users patch` - Update user details.

Partially updates a user resource by applying the supplied operations on specific user attributes.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

### `bricks account users update` - Replace a user.

Replaces a user's information with the data supplied in request.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--active` - If this user is active.
 * `--display-name` - String that represents a concatenation of given and family names.
 * `--external-id` -
 * `--id` - Databricks user ID.
 * `--user-name` - Email address of the Databricks user.

## `bricks account vpc-endpoints` - Manage VPC endpoints.

These commands manage VPC endpoint configurations for this account.

### `bricks account vpc-endpoints create` - Create VPC endpoint configuration.

Creates a VPC endpoint configuration, which represents a
[VPC endpoint](https://docs.aws.amazon.com/vpc/latest/privatelink/vpc-endpoints.html)
object in AWS used to communicate privately with Databricks over
[AWS PrivateLink](https://aws.amazon.com/privatelink).

After you create the VPC endpoint configuration, the Databricks
[endpoint service](https://docs.aws.amazon.com/vpc/latest/privatelink/privatelink-share-your-services.html)
automatically accepts the VPC endpoint.

Before configuring PrivateLink, read the
[Databricks article about PrivateLink](https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html).

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body * `--aws-vpc-endpoint-id` - The ID of the VPC endpoint object in AWS.
 * `--region` - The AWS region in which this VPC endpoint object exists.

### `bricks account vpc-endpoints delete` - Delete VPC endpoint configuration.

Deletes a VPC endpoint configuration, which represents an
[AWS VPC endpoint](https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html) that
can communicate privately with Databricks over [AWS PrivateLink](https://aws.amazon.com/privatelink).

Before configuring PrivateLink, read the [Databricks article about PrivateLink](https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html).

### `bricks account vpc-endpoints get` - Get a VPC endpoint configuration.

Gets a VPC endpoint configuration, which represents a [VPC endpoint](https://docs.aws.amazon.com/vpc/latest/privatelink/concepts.html) object in AWS used to communicate privately with Databricks over
[AWS PrivateLink](https://aws.amazon.com/privatelink).

### `bricks account vpc-endpoints list` - Get all VPC endpoint configurations.

Gets a list of all VPC endpoints for an account, specified by ID.

Before configuring PrivateLink, read the [Databricks article about PrivateLink](https://docs.databricks.com/administration-guide/cloud-configurations/aws/privatelink.html).

## `bricks warehouses` - Manage Databricks SQL warehouses.

A SQL warehouse is a compute resource that lets you run SQL commands on data objects within
Databricks SQL. Compute resources are infrastructure resources that provide processing
capabilities in the cloud.

### `bricks warehouses create` - Create a warehouse.

Creates a new SQL warehouse.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--auto-stop-mins` - The amount of time in minutes that a SQL warehouse must be idle (i.e., no RUNNING queries) before it is automatically stopped.
 * `--cluster-size` - Size of the clusters allocated for this warehouse.
 * `--creator-name` - warehouse creator name.
 * `--enable-photon` - Configures whether the warehouse should use Photon optimized clusters.
 * `--enable-serverless-compute` - Configures whether the warehouse should use serverless compute.
 * `--instance-profile-arn` - Deprecated.
 * `--max-num-clusters` - Maximum number of clusters that the autoscaler will create to handle concurrent queries.
 * `--min-num-clusters` - Minimum number of available clusters that will be maintained for this SQL warehouse.
 * `--name` - Logical name for the cluster.
 * `--spot-instance-policy` - Configurations whether the warehouse should use spot instances.
 * `--warehouse-type` - Warehouse type: `PRO` or `CLASSIC`.

### `bricks warehouses delete` - Delete a warehouse.

Deletes a SQL warehouse.

Flags:
 * `--no-wait` - do not wait to reach DELETED state.
 * `--timeout` - maximum amount of time to reach DELETED state.

### `bricks warehouses edit` - Update a warehouse.

Updates the configuration for a SQL warehouse.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--auto-stop-mins` - The amount of time in minutes that a SQL warehouse must be idle (i.e., no RUNNING queries) before it is automatically stopped.
 * `--cluster-size` - Size of the clusters allocated for this warehouse.
 * `--creator-name` - warehouse creator name.
 * `--enable-photon` - Configures whether the warehouse should use Photon optimized clusters.
 * `--enable-serverless-compute` - Configures whether the warehouse should use serverless compute.
 * `--instance-profile-arn` - Deprecated.
 * `--max-num-clusters` - Maximum number of clusters that the autoscaler will create to handle concurrent queries.
 * `--min-num-clusters` - Minimum number of available clusters that will be maintained for this SQL warehouse.
 * `--name` - Logical name for the cluster.
 * `--spot-instance-policy` - Configurations whether the warehouse should use spot instances.
 * `--warehouse-type` - Warehouse type: `PRO` or `CLASSIC`.

### `bricks warehouses get` - Get warehouse info.

Gets the information for a single SQL warehouse.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.

### `bricks warehouses get-workspace-warehouse-config` - Get the workspace configuration.

Gets the workspace level configuration that is shared by all SQL warehouses in a workspace.

### `bricks warehouses list` - List warehouses.

Lists all SQL warehouses that a user has manager permissions on.

Flags:
 * `--run-as-user-id` - Service Principal which will be used to fetch the list of warehouses.

### `bricks warehouses set-workspace-warehouse-config` - Set the workspace configuration.

Sets the workspace level configuration that is shared by all SQL warehouses in a workspace.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--google-service-account` - GCP only: Google Service Account used to pass to cluster to access Google Cloud Storage.
 * `--instance-profile-arn` - AWS Only: Instance profile used to pass IAM role to the cluster.
 * `--security-policy` - Security policy for warehouses.
 * `--serverless-agreement` - Internal.

### `bricks warehouses start` - Start a warehouse.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.

### `bricks warehouses stop` - Stop a warehouse.

Flags:
 * `--no-wait` - do not wait to reach STOPPED state.
 * `--timeout` - maximum amount of time to reach STOPPED state.

## `bricks workspace` - The Workspace API allows you to list, import, export, and delete notebooks and folders.

A notebook is a web-based interface to a document that contains runnable code, visualizations, and explanatory text.

### `bricks workspace delete` - Delete a workspace object.

Delete a workspace object.

Deletes an object or a directory (and optionally recursively deletes all objects in the directory).
* If `path` does not exist, this call returns an error `RESOURCE_DOES_NOT_EXIST`.
* If `path` is a non-empty directory and `recursive` is set to `false`, this call returns an error `DIRECTORY_NOT_EMPTY`.

Object deletion cannot be undone and deleting a directory recursively is not atomic.

Flags:
 * `--recursive` - The flag that specifies whether to delete the object recursively.

### `bricks workspace export` - Export a workspace object.

Exports an object or the contents of an entire directory.

If `path` does not exist, this call returns an error `RESOURCE_DOES_NOT_EXIST`.

One can only export a directory in `DBC` format. If the exported data would exceed size limit, this call returns `MAX_NOTEBOOK_SIZE_EXCEEDED`. Currently, command does not support exporting a library.

Flags:
 * `--direct-download` - Flag to enable direct download.
 * `--format` - This specifies the format of the exported file.

### `bricks workspace get-status` - Get status.

Gets the status of an object or a directory.
If `path` does not exist, this call returns an error `RESOURCE_DOES_NOT_EXIST`.

### `bricks workspace import` - Import a workspace object.

Imports a workspace object (for example, a notebook or file) or the contents of an entire directory.
If `path` already exists and `overwrite` is set to `false`, this call returns an error `RESOURCE_ALREADY_EXISTS`.
One can only use `DBC` format to import a directory.

Flags:
 * `--content` - The base64-encoded content.
 * `--format` - This specifies the format of the file to be imported.
 * `--language` - The language of the object.
 * `--overwrite` - The flag that specifies whether to overwrite existing object.

### `bricks workspace list` - List contents.

Lists the contents of a directory, or the object if it is not a directory.If
the input path does not exist, this call returns an error `RESOURCE_DOES_NOT_EXIST`.

Flags:
 * `--notebooks-modified-after` - ...

### `bricks workspace mkdirs` - Create a directory.

Creates the specified directory (and necessary parent directories if they do not exist).
If there is an object (not a directory) at any prefix of the input path, this call returns
an error `RESOURCE_ALREADY_EXISTS`.

Note that if this operation fails it may have succeeded in creating some of the necessary parrent directories.

## `bricks account workspace-assignment` - The Workspace Permission Assignment API allows you to manage workspace permissions for principals in your account.

The Workspace Permission Assignment API allows you to manage workspace permissions for principals in your account.

### `bricks account workspace-assignment delete` - Delete permissions assignment.

Deletes the workspace permissions assignment in a given account and workspace for the specified principal.

### `bricks account workspace-assignment get` - List workspace permissions.

Get an array of workspace permissions for the specified account and workspace.

### `bricks account workspace-assignment list` - Get permission assignments.

Get the permission assignments for the specified Databricks Account and Databricks Workspace.

### `bricks account workspace-assignment update` - Create or update permissions assignment.

Creates or updates the workspace permissions assignment in a given account and workspace for the specified principal.

Flags:
 * `--json` - either inline JSON string or @path/to/file.json with request body

## `bricks workspace-conf` - command allows updating known workspace settings for advanced users.

command allows updating known workspace settings for advanced users.

### `bricks workspace-conf get-status` - Check configuration status.

Gets the configuration status for a workspace.

### `bricks workspace-conf set-status` - Enable/disable features.

Sets the configuration status for a workspace, including enabling or disabling it.

## `bricks account workspaces` - These commands manage workspaces for this account.

These commands manage workspaces for this account. A Databricks workspace is an environment for
accessing all of your Databricks assets. The workspace organizes objects (notebooks,
libraries, and experiments) into folders, and provides access to data and computational
resources such as clusters and jobs.

These endpoints are available if your account is on the E2 version of the platform or on
a select custom plan that allows multiple workspaces per account.

### `bricks account workspaces create` - Create a new workspace.

Creates a new workspace.

**Important**: This operation is asynchronous. A response with HTTP status code 200 means
the request has been accepted and is in progress, but does not mean that the workspace
deployed successfully and is running. The initial workspace status is typically
`PROVISIONING`. Use the workspace ID (`workspace_id`) field in the response to identify
the new workspace and make repeated `GET` requests with the workspace ID and check
its status. The workspace becomes available when the status changes to `RUNNING`.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.
 * `--json` - either inline JSON string or @path/to/file.json with request body
 * `--aws-region` - The AWS region of the workspace's data plane.
 * `--cloud` - The cloud provider which the workspace uses.
 * `--credentials-id` - ID of the workspace's credential configuration object.
 * `--deployment-name` - The deployment name defines part of the subdomain for the workspace.
 * `--location` - The Google Cloud region of the workspace data plane in your Google account.
 * `--managed-services-customer-managed-key-id` - The ID of the workspace's managed services encryption key configuration object.
 * `--network-id` -
 * `--pricing-tier` - The pricing tier of the workspace.
 * `--private-access-settings-id` - ID of the workspace's private access settings object.
 * `--storage-configuration-id` - The ID of the workspace's storage configuration object.
 * `--storage-customer-managed-key-id` - The ID of the workspace's storage encryption key configuration object.

### `bricks account workspaces delete` - Delete a workspace.

Terminates and deletes a Databricks workspace. From an API perspective, deletion is immediate. However, it might take a few minutes for all workspaces resources to be deleted, depending on the size and number of workspace resources.

This operation is available only if your account is on the E2 version of the platform or on a select custom plan that allows multiple workspaces per account.

### `bricks account workspaces get` - Get a workspace.

Gets information including status for a Databricks workspace, specified by ID. In the response, the `workspace_status` field indicates the current status. After initial workspace creation (which is asynchronous), make repeated `GET` requests with the workspace ID and check its status. The workspace becomes available when the status changes to `RUNNING`.

For information about how to create a new workspace with command **including error handling**, see [Create a new workspace using the Account API](http://docs.databricks.com/administration-guide/account-api/new-workspace.html).

This operation is available only if your account is on the E2 version of the platform or on a select custom plan that allows multiple workspaces per account.

### `bricks account workspaces list` - Get all workspaces.

Gets a list of all workspaces associated with an account, specified by ID.

This operation is available only if your account is on the E2 version of the platform or on a select custom plan that allows multiple workspaces per account.

### `bricks account workspaces update` - Update workspace configuration.

Updates a workspace configuration for either a running workspace or a failed workspace. The elements that can be updated varies between these two use cases.

Update a failed workspace:
You can update a Databricks workspace configuration for failed workspace deployment for some fields, but not all fields. For a failed workspace, this request supports updates to the following fields only:
- Credential configuration ID
- Storage configuration ID
- Network configuration ID. Used only to add or change a network configuration for a customer-managed VPC. For a failed workspace only, you can convert a workspace with Databricks-managed VPC to use a customer-managed VPC by adding this ID. You cannot downgrade a workspace with a customer-managed VPC to be a Databricks-managed VPC. You can update the network configuration for a failed or running workspace to add PrivateLink support, though you must also add a private access settings object.
- Key configuration ID for managed services (control plane storage, such as notebook source and Databricks SQL queries). Used only if you use customer-managed keys for managed services.
- Key configuration ID for workspace storage (root S3 bucket and, optionally, EBS volumes). Used only if you use customer-managed keys for workspace storage. **Important**: If the workspace was ever in the running state, even if briefly before becoming a failed workspace, you cannot add a new key configuration ID for workspace storage.
- Private access settings ID to add PrivateLink support. You can add or update the private access settings ID to upgrade a workspace to add support for front-end, back-end, or both types of connectivity. You cannot remove (downgrade) any existing front-end or back-end PrivateLink support on a workspace.

After calling the `PATCH` operation to update the workspace configuration, make repeated `GET` requests with the workspace ID and check the workspace status. The workspace is successful if the status changes to `RUNNING`.

For information about how to create a new workspace with command **including error handling**, see [Create a new workspace using the Account API](http://docs.databricks.com/administration-guide/account-api/new-workspace.html).

Update a running workspace:
You can update a Databricks workspace configuration for running workspaces for some fields, but not all fields. For a running workspace, this request supports updating the following fields only:
- Credential configuration ID

- Network configuration ID. Used only if you already use a customer-managed VPC. You cannot convert a running workspace from a Databricks-managed VPC to a customer-managed VPC. You can use a network configuration update in command for a failed or running workspace to add support for PrivateLink, although you also need to add a private access settings object.

- Key configuration ID for managed services (control plane storage, such as notebook source and Databricks SQL queries). Databricks does not directly encrypt the data with the customer-managed key (CMK). Databricks uses both the CMK and the Databricks managed key (DMK) that is unique to your workspace to encrypt the Data Encryption Key (DEK). Databricks uses the DEK to encrypt your workspace's managed services persisted data. If the workspace does not already have a CMK for managed services, adding this ID enables managed services encryption for new or updated data. Existing managed services data that existed before adding the key remains not encrypted with the DEK until it is modified. If the workspace already has customer-managed keys for managed services, this request rotates (changes) the CMK keys and the DEK is re-encrypted with the DMK and the new CMK.
- Key configuration ID for workspace storage (root S3 bucket and, optionally, EBS volumes). You can set this only if the workspace does not already have a customer-managed key configuration for workspace storage.
- Private access settings ID to add PrivateLink support. You can add or update the private access settings ID to upgrade a workspace to add support for front-end, back-end, or both types of connectivity. You cannot remove (downgrade) any existing front-end or back-end PrivateLink support on a workspace.

**Important**: To update a running workspace, your workspace must have no running compute resources that run in your workspace's VPC in the Classic data plane. For example, stop all all-purpose clusters, job clusters, pools with running clusters, and Classic SQL warehouses. If you do not terminate all cluster instances in the workspace before calling command, the request will fail.

**Important**: Customer-managed keys and customer-managed VPCs are supported by only some deployment types and subscription types. If you have questions about availability, contact your Databricks representative.

This operation is available only if your account is on the E2 version of the platform or on a select custom plan that allows multiple workspaces per account.

Flags:
 * `--no-wait` - do not wait to reach RUNNING state.
 * `--timeout` - maximum amount of time to reach RUNNING state.
 * `--aws-region` - The AWS region of the workspace's data plane (for example, `us-west-2`).
 * `--credentials-id` - ID of the workspace's credential configuration object.
 * `--managed-services-customer-managed-key-id` - The ID of the workspace's managed services encryption key configuration object.
 * `--network-id` - The ID of the workspace's network configuration object.
 * `--storage-configuration-id` - The ID of the workspace's storage configuration object.
 * `--storage-customer-managed-key-id` - The ID of the key configuration object for workspace storage.
