## /Workspace prefix

This PR (https://github.com/databricks/cli/pull/1724) introduced the following changes:

1. Default remote bundle paths became automatically prefixed with /Workspace prefix
2. All usage of path strings like `/Workspace/${workspace.root_path}/...` and etc. in bundle configuration now automatically replaced with `${workspace.root_path}/...` and generates a warning as part of `bundle validate`

If users have specified a custom `workspace.{root,artifact,file,state}_path`, then DABs will automatically ensure it is prefixed with /Workspace.

This also means that if they use any of these as variables (for example, `my_config_path: /Workspace/${workspace.file_path}/config`), then they’ll need to update those entries to remove the `/Workspace` prefix to avoid the warning.

If they pass along one of these as variables and prefix them in their code, they’ll need to update their code to no longer do this.

The reason for this is the following:

Historically, the workspace file system was rooted at `/`, with home directories under `/Users`, for example.
To access workspace paths through the API you would use these paths directly. To access workspace paths from your code, you could use the `/Workspace` file path that mounts the workspace file system on Databricks clusters. Home directories were available under `/Workspace/Users`.  This duality proved inconvenient, and with the introduction of UC volumes it even became ambiguous (UC volumes are accessible under `/Volumes` ). To avoid both the duality of workspace paths and as well as ambiguity between workspace paths and UC volume paths, we’re now prefixing all workspace paths with `/Workspace`. This means they no longer need to be manually prefixed to be used as a POSIX path (e.g. to open a file from Python), nor is there ambiguity between workspace paths and UC volume paths.
