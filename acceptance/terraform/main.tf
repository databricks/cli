terraform {
  required_providers {
    databricks = {
      source  = "databricks/databricks"
      version = "1.85.0"
    }
  }

  required_version = "= 1.5.5"
}

provider "databricks" {
  # Optionally, specify the Databricks host and token
  # host  = "https://<your-databricks-instance>"
  # token = "<YOUR_PERSONAL_ACCESS_TOKEN>"
}

data "databricks_current_user" "me" {
  # Retrieves the current user's information
}

output "username" {
  description = "Username"
  value       = "${data.databricks_current_user.me.user_name}"
}
