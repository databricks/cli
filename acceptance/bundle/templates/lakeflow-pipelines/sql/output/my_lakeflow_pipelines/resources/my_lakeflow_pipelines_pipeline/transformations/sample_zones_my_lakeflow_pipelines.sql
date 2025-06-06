-- This file defines a sample transformation.
-- Edit the sample below or add new transformations
-- using "+ Add" in the file browser.

CREATE MATERIALIZED VIEW sample_zones_my_lakeflow_pipelines AS
SELECT
    pickup_zip,
    SUM(fare_amount) AS total_fare
FROM sample_trips_my_lakeflow_pipelines
GROUP BY pickup_zip
