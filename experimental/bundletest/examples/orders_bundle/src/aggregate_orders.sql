-- The real deployed artifact for the aggregate_orders job.
-- Silver -> gold: one summary row over the cleaned orders.
CREATE OR REPLACE TABLE shop.gold.order_summary AS
SELECT COUNT(*) AS order_count, SUM(total_price) AS total_revenue
FROM shop.silver.orders;
