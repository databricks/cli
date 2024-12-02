import dlt
from pyspark.sql import DataFrame
from databricks.sdk.runtime import spark


@dlt.view(comment="Small set of taxis for development (uses LIMIT 10)")
def taxis() -> DataFrame:
    return spark.sql("SELECT * FROM samples.nyctaxi.trips LIMIT 10")
