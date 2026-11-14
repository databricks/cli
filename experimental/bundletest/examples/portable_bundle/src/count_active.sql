-- Distinct users with a login event. Portable SQL: runs identically on DuckDB and
-- Databricks, so the test that exercises it means the same thing on both tiers.
CREATE OR REPLACE TABLE demo.gold.active_users AS
SELECT DISTINCT user_id
FROM demo.bronze.events
WHERE user_id IS NOT NULL AND event = 'login';
