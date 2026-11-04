-- bronze -> silver: drop duplicate/null-id rows and pin the price to a fixed decimal.
CREATE OR REPLACE TABLE main.bundletest_cloud.orders AS
SELECT DISTINCT order_id, CAST(total_price AS DECIMAL(10, 2)) AS total_price
FROM main.bundletest_cloud.raw_orders
WHERE order_id IS NOT NULL;
