"""Vector search index wiring — its endpoint and the table it syncs from.

CAN test locally:
- the index exists, its endpoint, and the source table it delta-syncs from
- that it syncs the gold table the jobs produce (cross-resource wiring)

CANNOT test locally — needs the cloud backend:
- building / querying the index
"""


def test_index_syncs_gold_table(env):
    index = env.vector_search_index("orders_index")
    assert index.exists()
    assert index.endpoint_name == "orders-vs-endpoint"
    assert index.source_table() == "shop.gold.order_summary"
