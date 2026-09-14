"""Quality monitor wiring — which table the monitor is attached to.

CAN test locally:
- the monitor exists and the table it monitors
- that it monitors the gold table the jobs produce (cross-resource wiring)

CANNOT test locally — needs the cloud backend:
- refreshing the monitor / reading its computed metrics
"""


def test_monitor_targets_gold_table(env):
    monitor = env.quality_monitor("orders_quality")
    assert monitor.exists()
    assert monitor.monitored_table() == "shop.gold.order_summary"
