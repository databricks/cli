create or refresh materialized view double as
select
    `id`,
    `id` * 2 as double
from
    LIVE.range
