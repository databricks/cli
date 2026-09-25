-- The real deployed artifact for the transform_orders job.
-- Bronze -> silver: drop duplicate/null-id rows and pin the price to a fixed decimal.
CREATE OR REPLACE TABLE shop.silver.orders AS
SELECT DISTINCT order_id, CAST(total_price AS DECIMAL(10, 2)) AS total_price
FROM shop.bronze.raw_orders
WHERE order_id IS NOT NULL;
