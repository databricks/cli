-- This query is executed using Databricks Workflows (see resources/default_sql_sql_job.yml)

CREATE OR REPLACE VIEW
  IDENTIFIER(CONCAT({{catalog}}, '.', {{schema}}, '.', 'orders_daily'))
AS SELECT
  order_date, count(*) AS number_of_orders
FROM
  IDENTIFIER(CONCAT({{catalog}}, '.', {{schema}}, '.', 'orders_raw'))

-- During development, only process a smaller range of data
WHERE {{bundle_target}} == "prod" OR (order_date >= '2019-08-01' AND order_date < '2019-09-01')

GROUP BY order_date
