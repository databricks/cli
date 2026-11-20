-- silver -> gold: one summary row over the cleaned orders.
CREATE OR REPLACE TABLE main.bundletest_cloud.order_summary AS
SELECT COUNT(*) AS order_count, SUM(total_price) AS total_revenue
FROM main.bundletest_cloud.orders;
