-- This file defines a sample transformation.
-- Edit the sample below or add new transformations
-- using "+ Add" in the file browser.

CREATE MATERIALIZED VIEW sample_users_my_lakeflow_pipelines AS
SELECT
    user_id,
    email,
    name,
    user_type
FROM samples.wanderbricks.users
