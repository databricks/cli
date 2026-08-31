Fixed deploying a change to a Lakebase map field such as
`postgres_endpoints.settings.pg_settings`. The update masked the individual map entry,
which the API rejects with `Unknown field path in update_mask`; a map or repeated field
is addressable only as a whole.
