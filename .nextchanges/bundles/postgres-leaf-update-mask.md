Fixed deploying a change to a field nested inside a Lakebase message, such as
`postgres_projects.default_endpoint_settings.autoscaling_limit_max_cu`, when the bundle
declares no suspension field. The update masked the enclosing message as well as the
field, and the API then rejected the request for the missing `spec.default_endpoint_settings.suspension`.
