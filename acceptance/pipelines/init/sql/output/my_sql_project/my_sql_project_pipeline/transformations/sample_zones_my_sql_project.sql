-- This file defines a sample transformation.
-- Edit the sample below or add new transformations
-- using "+ Add" in the file browser.

CREATE MATERIALIZED VIEW sample_zones_my_sql_project AS
SELECT
    pickup_zip,
    SUM(fare_amount) AS total_fare
FROM sample_trips_my_sql_project
GROUP BY pickup_zip
