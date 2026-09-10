"""Genie Space wiring — which tables the space's datasets read from.

A Genie Space carries the same kind of inline serialized definition as a dashboard, so
source_tables() reuses the same parser.

CAN test locally:
- the source tables a space with an inline serialized_space reads from

CANNOT test locally — needs the cloud backend:
- a space defined only by file_path -> skips loudly (no inline queries in databricks.yml)
- asking the space a question / running its queries
"""


def test_genie_space_reads_gold_table(env):
    space = env.genie_space("orders_genie")
    assert space.exists()
    assert space.source_tables() == ["shop.gold.order_summary"]
