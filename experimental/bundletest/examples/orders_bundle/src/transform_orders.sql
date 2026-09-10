-- The real deployed artifact for the transform_orders job.
-- Bronze -> silver: drop duplicate rows and rows with a null order_id.
CREATE OR REPLACE TABLE shop.silver.orders AS
SELECT DISTINCT order_id, total_price
FROM shop.bronze.raw_orders
WHERE order_id IS NOT NULL;
