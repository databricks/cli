import dlt
from pyspark.sql.functions import to_date, count
from pyspark.sql import DataFrame


@dlt.table(comment="Daily statistics of NYC Taxi trips")
def taxi_stats() -> DataFrame:
    """Read from the 'taxis' view from etl_pipeline/sources."""
    taxis = dlt.read("taxis")

    return filter_taxis(taxis)


def filter_taxis(taxis: DataFrame) -> DataFrame:
    """Group by date and calculate the number of trips."""
    return (
        taxis.withColumn("pickup_date", to_date("tpep_pickup_datetime"))
        .groupBy("pickup_date")
        .agg(count("*").alias("number_of_trips"))
    )
