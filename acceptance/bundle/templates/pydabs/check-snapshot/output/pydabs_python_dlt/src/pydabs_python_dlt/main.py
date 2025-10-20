from databricks.sdk.runtime import spark
from pyspark.sql import DataFrame


def find_all_taxis() -> DataFrame:
    return spark.read.table("samples.nyctaxi.trips")


def main():
    find_all_taxis().show(5)


if __name__ == "__main__":
    main()
