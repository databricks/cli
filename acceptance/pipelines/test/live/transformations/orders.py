from pyspark import pipelines as dp


@dp.materialized_view
def orders():
    return spark.range(3)
