import dlt
from pyspark.sql import SparkSession

spark = SparkSession.builder.getOrCreate()


@dlt.table
def table1():
    return spark.range(10)


@dlt.table
def table2():
    return spark.range(20)
