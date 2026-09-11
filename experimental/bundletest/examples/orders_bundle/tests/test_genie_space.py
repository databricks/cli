"""Genie Space wiring — which tables the space is grounded on.

A Genie space declares its tables explicitly under data_sources.tables[].identifier — a
different serialized schema from a Lakeview dashboard, so it has its own parser.

CAN test locally:
- the source tables a space with an inline serialized_space declares

CANNOT test locally — needs the cloud backend:
- a space defined only by file_path -> skips loudly (no inline definition in databricks.yml)
- asking the space a question / running its queries
"""


def test_genie_space_reads_gold_table(env):
    space = env.genie_space("orders_genie")
    assert space.exists()
    assert space.source_tables() == ["shop.gold.order_summary"]
