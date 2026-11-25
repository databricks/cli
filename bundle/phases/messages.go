package phases

// DataLossWarning is the warning shown when a non-interactive command is about
// to delete data-bearing resources.
const DataLossWarning = "Deleting data assets such as schemas, pipelines, or volumes may cause permanent data loss and should be carefully reviewed."

// TerraformStateRemovedMessage is the error text shown when a command that does not
// migrate on its own (bind, unbind) finds an existing Terraform state. The Terraform
// deployment engine was removed in v1.20.0; "bundle deploy" migrates the state to the
// direct engine or the Terraform engine must be used by running an older CLI version.
const TerraformStateRemovedMessage = `this deployment's state uses the Terraform engine, which has been removed in Databricks CLI v1.20.0

Run "databricks bundle deploy" to migrate the state to the direct engine, or install Databricks CLI v1.19.x`
