// This file is used to build the sbt project with Databricks Connect.
// This also includes the instructions on how to to create the jar uploaded via databricks bundle
scalaVersion := "2.13.16"

name := "my_default_scala"
organization := "com.examples"
version := "0.1"

libraryDependencies += "com.databricks" %% "databricks-connect" % "17.0.+"
libraryDependencies += "org.slf4j" % "slf4j-simple" % "2.0.16"

libraryDependencies += "org.scalatest" %% "scalatest" % "3.2.19" % Test

assembly / assemblyOption ~= { _.withIncludeScala(false) }
assembly / assemblyExcludedJars := {
  val cp = (assembly / fullClasspath).value
  cp filter { _.data.getName.matches("scala-.*") } // remove Scala libraries
}

assemblyMergeStrategy := {
  case _ => MergeStrategy.preferProject
}

// to run with new jvm options, a fork is required otherwise it uses same options as sbt process
fork := true
javaOptions += "--add-opens=java.base/java.nio=ALL-UNNAMED"

// To ensure logs are written to System.out by default and not System.err
javaOptions += "-Dorg.slf4j.simpleLogger.logFile=System.out"
