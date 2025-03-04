package com.examples

import org.apache.spark.sql.{Dataset, Row, SparkSession}

class NycTaxi(val spark: SparkSession) {
  /**
   * Return the first rows of the NYC taxi trips table.
   * By default, returns the first 10.
   * @param limit the number of rows to return
   * @return the result rows
   */
  def trips(limit: Int = 10): Dataset[Row]  = {
    spark.read.table("samples.nyctaxi.trips").limit(limit)
  }

  /**
   * Count the total number of trips.
   * When pickupZip is provided, count trips for started from that zip code.
   * @param pickupZip optionally filter by trips starting from this zip code.
   * @return total number of trips.
   */
  def countTrips(pickupZip: Option[String] = None): Long = {
    val df = spark.read.table("samples.nyctaxi.trips")
    val byPickup = pickupZip match {
      case Some(x) => df.filter(s"pickup_zip = '$x'")
      case None => df
    }
    byPickup.count()
  }
}

