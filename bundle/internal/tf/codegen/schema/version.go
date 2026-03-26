package schema

const ProviderVersion = "1.111.0"

// Checksums for the Databricks Terraform provider archive. These are not used
// inside the CLI. They are co-located here to be output in the
// "databricks bundle debug terraform" output. Downstream applications like the
// CLI docker image use these checksums to verify the integrity of the downloaded
// provider archive. Please update these when the provider version is bumped.
// The checksums are obtained from https://github.com/databricks/terraform-provider-databricks/releases.
const ProviderChecksumLinuxAmd64 = "c1b46bbaf5c4a0b253309dad072e05025e24731536719d4408bacd48dc0ccfd9"
const ProviderChecksumLinuxArm64 = "ce379c424009b01ec4762dee4d0db27cfc554d921b55a0af8e4203b3652259e9"
