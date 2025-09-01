import dlt
from pyspark.sql.functions import col
from default_python_etl.utilities import utils


# This file defines a sample transformation.
# Edit the sample below or add new transformations
# using "+ Add" in the file browser.


@dlt.table
def sample_trips_jan_01_1034():
    return spark.read.table("samples.nyctaxi.trips").withColumn(
        "trip_distance_km", utils.distance_km(col("trip_distance"))
    )
