-- Example materialized view
-- Read more about materialized at https://docs.getdbt.com/reference/resource-configs/databricks-configs#materialized-views-and-streaming-tables
-- Current limitation: a "full refresh" is needed in case the definition below is changed; see https://github.com/databricks/dbt-databricks/issues/561.
{{ config(materialized = 'materialized_view') }}


select order_date, count(*) AS number_of_orders
from {{ ref('orders_raw') }}
where order_date is not null
group by order_date
