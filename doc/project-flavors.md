Project Flavors
---

`bricks` CLI detects variout project flavors dynamically every run, though sometimes you may be interested in overriding the defaults.

## Maven

If there's a `pom.xml` file in the same folder as [`databricks.yml`](configuration.md), `mvn clean package` is invoked during [`build`](project-lifecycle.md#build) stage, followed by uploading `target/$artifactId-$version.jar` file to DBFS during the [`upload`](project-lifecycle.md#upload) stage, installing it as a library on [Development Cluster](configuration.md#development-cluster) and waiting for the installation to succeed, reporting the error back otherwise.

## Python

If there's a `setup.py` file in the [project root](configuration.md), ...

