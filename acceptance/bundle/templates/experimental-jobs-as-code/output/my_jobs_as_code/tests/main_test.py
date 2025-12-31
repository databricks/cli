from my_jobs_as_code.main import get_taxis, get_spark

# running tests requires installing databricks-connect, e.g. by uncommenting it in pyproject.toml


def test_main():
    taxis = get_taxis(get_spark())
    assert taxis.count() > 5
