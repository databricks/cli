from databricks.sdk.runtime import spark
from pyspark.sql import DataFrame
from my_default_python import users


def test_find_all_users():
    results = users.find_all_users()
    assert results.count() > 5
