"""Job logic for the in-memory backend.

Each entry in JOBS maps a bundle job name to a callable that receives ``sql`` — a
function that runs one statement. On the cloud backend the real deployed job runs
instead; this stands in for it locally so a transform can be exercised with zero infra.
"""

JOBS = {}


def _job(name):
    def register(fn):
        JOBS[name] = fn
        return fn

    return register


@_job("transform_orders")
def transform_orders(sql):
    """Bronze -> silver: drop duplicate and null-id rows."""
    sql(
        "CREATE TABLE silver.orders AS "
        "SELECT DISTINCT order_id, total_price "
        "FROM bronze.raw_orders "
        "WHERE order_id IS NOT NULL"
    )
