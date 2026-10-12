import dlt
from pyspark.sql import SparkSession


@dlt.table
def my_table():
    return spark.range(10)
