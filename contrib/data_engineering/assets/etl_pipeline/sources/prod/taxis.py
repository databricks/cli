import dlt
from pyspark.sql import DataFrame
from databricks.sdk.runtime import spark


@dlt.view
def taxis() -> DataFrame:
    return spark.sql("SELECT * FROM samples.nyctaxi.trips")
