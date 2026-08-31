Fixed deploying a change to a field nested inside a Lakebase message, such as
`postgres_projects.default_endpoint_settings.autoscaling_limit_max_cu`, when the bundle
declares no suspension field ([#6440](https://github.com/databricks/cli/pull/6440)).
