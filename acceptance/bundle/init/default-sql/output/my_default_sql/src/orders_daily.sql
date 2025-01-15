-- This query is executed using Databricks Workflows (see resources/my_default_sql_sql.job.yml)

USE CATALOG {{catalog}};
USE IDENTIFIER({{schema}});

CREATE OR REPLACE MATERIALIZED VIEW
  orders_daily
AS SELECT
  order_date, count(*) AS number_of_orders
FROM
  orders_raw

WHERE if(
  {{bundle_target}} = "prod",
  true,

  -- During development, only process a smaller range of data
  order_date >= '2019-08-01' AND order_date < '2019-09-01'
)

GROUP BY order_date
