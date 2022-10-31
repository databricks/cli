Project Lifecycle
---

Project lifecycle consists of different execution phases. This document aims at describing them as toroughly as possible.

## `init`

`bricks init` creates a [`databricks.yml`](configuration.md) file in the directory, where `bricks` CLI was invoked. It walks you through the interactive command prompts. The goal of this stage is to setup project flavor and connectivity to a Databricks workspapce.

## `prepare`

`bricks prepare` prepares the local filesystem for the following lifecycle stages, like rolling out the relevant Virtual Environment for [Python projects](project-flavors.md#python).

## `build`

`bricks build` triggers the relevant commands to package artifacts, like Java or Scala [JARs](project-flavors.md#maven) or Python [Wheels](project-flavors.md#python). It's also possible to have a multi-flavor project, like [Mosaic](https://github.com/databrickslabs/mosaic), where built Wheel depends on a built JAR.

## `upload`

`bricks upload` takes the artifacts created by [`bricks build`](#build) and uploads them to a path following configured [isolation level](configuration.md#isolation-levels).

## `deploy`

.. creates clusters

## `install`

`bricks install` takes remote paths created by [`bricks upload`](#upload) for artifacts created by [`bricks build`](#build) and installs them on [Development Cluster](configuration.md#development-cluster) following the configured [isolation level](configuration.md#isolation-levels).