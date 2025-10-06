from pyspark.sql.functions import udf
from pyspark.sql.types import FloatType


@udf(returnType=FloatType())
def distance_km(distance_miles):
    """Convert distance from miles to kilometers (1 mile = 1.60934 km)."""
    return distance_miles * 1.60934
