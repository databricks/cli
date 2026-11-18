from pyspark import pipelines as dp


@dp.table
def orders():
    return spark.range(10)
