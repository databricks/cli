from ..sources.dev.taxis import taxis
from ..transformations import taxi_stats


def test_taxi_stats():
    result = taxi_stats.filter_taxis(taxis())
    assert len(result.collect()) > 5
