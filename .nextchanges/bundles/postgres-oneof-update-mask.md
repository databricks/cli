Fixed deploying a change to a Lakebase field that belongs to a oneof —
`postgres_branches.expire_time` and `.ttl`, `postgres_endpoints.suspend_timeout_duration`,
and `postgres_projects.default_endpoint_settings.suspend_timeout_duration`. The update
masked each field under its own name, which the API rejects; it accepts only the oneof
group name.
