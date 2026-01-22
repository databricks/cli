import dlt


@dlt.table
def my_table():
    return spark.range(1)
