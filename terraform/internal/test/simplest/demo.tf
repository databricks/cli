terraform {
  required_providers {
    databricks = {
      source  = "databrickslabs/databricks"
    }
  }
}

provider "databricks" {
}

# This file cannot be used for tests until `InsecureSkipVerify` is exposed though env var
# data "databricks_current_user" "me" {}
# data "databricks_spark_version" "latest" {}
# data "databricks_node_type" "smallest" {
#   local_disk = true
# }

resource "databricks_notebook" "this" {
  path     = "/Users/me@example.com/Terraform"
  language = "PYTHON"
  content_base64 = base64encode(<<-EOT
    # created from ${abspath(path.module)}
    display(spark.range(10))
    EOT
  )
}

resource "databricks_job" "this" {
  name = "Terraform Demo (me@example.com)"

  new_cluster {
    num_workers   = 1
    spark_version = "10.0.1"
    node_type_id  = "i3.xlarge"
  }

  notebook_task {
    notebook_path = databricks_notebook.this.path
  }
}