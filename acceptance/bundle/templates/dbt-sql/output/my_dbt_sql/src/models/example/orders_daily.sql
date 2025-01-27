
-- This model file defines a materialized view called 'orders_daily'
--
-- Read more about materialized at https://docs.getdbt.com/reference/resource-configs/databricks-configs#materialized-views-and-streaming-tables
-- Current limitation: a "full refresh" is needed in case the definition below is changed; see https://github.com/databricks/dbt-databricks/issues/561.
{{ config(materialized = 'materialized_view') }}

select order_date, count(*) AS number_of_orders

from {{ ref('orders_raw') }}

-- During development, only process a smaller range of data
{% if target.name != 'prod' %}
where order_date >= '2019-08-01' and order_date < '2019-09-01'
{% endif %}

group by order_date
