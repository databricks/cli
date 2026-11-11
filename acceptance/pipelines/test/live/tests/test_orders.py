# order_helpers is not on the import path; only its installation as a test dependency provides it.
from order_helpers import order_total


def test_order_total():
    assert order_total([1, 2, 3]) == 6
