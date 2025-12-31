from pyspark.sql import SparkSession, DataFrame


def get_users(spark: SparkSession) -> DataFrame:
    return spark.read.table("samples.wanderbricks.users")


# Create a new Databricks Connect session. If this fails,
# check that you have configured Databricks Connect correctly.
# See https://docs.databricks.com/dev-tools/databricks-connect.html.
def get_spark() -> SparkSession:
    try:
        from databricks.connect import DatabricksSession

        return DatabricksSession.builder.getOrCreate()
    except ImportError:
        return SparkSession.builder.getOrCreate()


def main():
    get_users(get_spark()).show(5)


if __name__ == "__main__":
    main()
