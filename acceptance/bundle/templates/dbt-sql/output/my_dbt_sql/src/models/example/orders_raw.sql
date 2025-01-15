-- This model file defines a streaming table called 'orders_raw'
--
-- The streaming table below ingests all JSON files in /databricks-datasets/retail-org/sales_orders/
-- Read more about streaming tables at https://docs.getdbt.com/reference/resource-configs/databricks-configs#materialized-views-and-streaming-tables
-- Current limitation: a "full refresh" is needed in case the definition below is changed; see https://github.com/databricks/dbt-databricks/issues/561.
{{ config(materialized = 'streaming_table') }}

select
  customer_name,
  date(timestamp(from_unixtime(try_cast(order_datetime as bigint)))) as order_date,
  order_number
from stream read_files(
  "/databricks-datasets/retail-org/sales_orders/",
  format => "json",
  header => true
)
