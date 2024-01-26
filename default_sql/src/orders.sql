-- This query is executed using Databricks Workflows as defined in resources/default_sql_sql_job.yml.

CREATE OR REPLACE VIEW
  SYMBOL(CONCAT({{catalog}}, '.', {{schema}}, '.', 'orders'))
AS SELECT
  order_date, count(*) AS number_of_orders
FROM
  SYMBOL(CONCAT({{catalog}}, '.', {{schema}}, '.', 'raw_orders'))

-- During development, only process a smaller range of data
WHERE {{bundle_target}} == prod OR (order_date >= '2019-08-01' AND order_date < '2019-09-01')

GROUP BY order_date
