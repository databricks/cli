# Simple pipeline file for testing
import dlt


@dlt.table
def my_table():
    return spark.range(10)
