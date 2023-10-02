from databricks.connect import DatabricksSession
from pyspark.sql import SparkSession
from default_python import main

# Create a new Databricks Connect session. If this fails,
# check that you have configured Databricks Connect correctly.
# See https://docs.databricks.com/dev-tools/databricks-connect.html.

SparkSession.builder = DatabricksSession.builder
SparkSession.builder.getOrCreate()

def test_main():
    taxis = main.get_taxis()
    assert taxis.count() > 5
