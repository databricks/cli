Project Configuration
---

_Good implicit defaults is better than explicit complex configuration._

Regardless of current working directory, `bricks` finds project root with `databricks.yml` file up the directory tree. Technically, there might be couple of different Databricks Projects in the same Git repository, but the recommended scenario is to have just one `databricks.yml` in the root of Git repo.