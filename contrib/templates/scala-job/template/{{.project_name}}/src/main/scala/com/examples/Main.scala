/*
This project is a simple example of how to use the Databricks Connect Scala client to run on
serverless or on a Databricks cluster.
 */
package com.examples

import com.databricks.connect.DatabricksSession
import org.apache.spark.sql.{SparkSession, functions => F}
import org.apache.spark.sql.functions.udf

object Main {
  def main(args: Array[String]): Unit = {
    println("Hello, World!")

    val spark = getSession()
    println("Showing range ...")
    spark.range(3).show()

    println("Showing nyctaxi trips ...")
    val nycTaxi = new NycTaxi(spark)
    val df = nycTaxi.trips()

    // Define a simple UDF that formats the passenger count as a string
    val testudf = udf((count: String) => s"test: $count")

    // Apply the UDF to the passenger_count column
    val transformedDF = df.withColumn("testresult", testudf(F.col("dropoff_zip")))

    // Show the transformed DataFrame
    transformedDF.show()
  }

  def getSession(): SparkSession = {
    // Get DATABRICKS_RUNTIME_VERSION environment variable
    if (sys.env.contains("DATABRICKS_RUNTIME_VERSION")) {
      println("Running in a Databricks cluster")
      SparkSession.builder().getOrCreate()
    } else {
      println("Running outside Databricks")
      DatabricksSession.builder()
        .serverless()
        .addCompiledArtifacts(Main.getClass.getProtectionDomain.getCodeSource.getLocation.toURI)
        .getOrCreate()
    }
  }
}
