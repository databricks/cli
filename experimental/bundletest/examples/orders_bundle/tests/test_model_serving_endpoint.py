"""Model serving endpoint wiring — which models it serves.

CAN test locally:
- the endpoint exists and the model(s) it serves

CANNOT test locally — needs the cloud backend:
- querying the endpoint / checking it is READY
"""


def test_endpoint_serves_model(env):
    endpoint = env.model_serving_endpoint("orders_model_endpoint")
    assert endpoint.exists()
    assert endpoint.served_models() == ["shop.ml.orders_model"]
