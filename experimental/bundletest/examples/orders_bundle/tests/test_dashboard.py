"""Dashboard wiring — which tables the dashboard's datasets read from.

source_tables() parses the dashboard's inline serialized definition and pulls the qualified
table each dataset query reads. This catches a dashboard pointed at a stale or misspelled
table without opening a workspace.

CAN test locally:
- the source tables a dashboard with an inline serialized_dashboard reads from
- that it's wired to the gold table the jobs produce (cross-resource wiring)

CANNOT test locally — needs the cloud backend:
- a dashboard defined only by file_path -> skips loudly (no inline queries in databricks.yml)
- that the dashboard renders / its queries actually run
"""


def test_dashboard_reads_gold_table(env):
    dashboard = env.dashboard("orders_overview")
    assert dashboard.exists()
    # aggregate_orders.sql writes shop.gold.order_summary — the dashboard reads it.
    assert dashboard.source_tables() == ["shop.gold.order_summary"]
