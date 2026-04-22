// Package terraform is the terraform-engine wrapper for ucm. It sits on top
// of the ucm/deploy/terraform/tfdyn converter (U3) and the ucm/deploy/lock
// Locker (U2), mirroring the shape of bundle/deploy/terraform without
// importing from bundle.
//
// Production code drives hashicorp/terraform-exec directly; tests inject a
// fake tfRunner so the full init/plan/apply/destroy surface is exercised
// without installing a real terraform binary.
package terraform

import (
	"context"

	"github.com/hashicorp/go-version"
)

// MainConfigFileName is the on-disk name of the generated Terraform JSON
// configuration written by Render. Mirrors bundle/deploy/terraform's
// TerraformConfigFileName (bundle.tf.json) — the "main.tf.json" name is
// chosen to match terraform's own convention for auto-loaded JSON config.
const MainConfigFileName = "main.tf.json"

// PlanFileName is the on-disk name of the plan artefact produced by Plan and
// consumed by Apply. Kept identical to bundle/deploy/terraform so reviewers
// have one fewer divergence to remember.
const PlanFileName = "plan"

// ProviderSource is the fully-qualified source address of the databricks
// terraform provider. Written into the `terraform.required_providers` block
// so `terraform init` resolves the provider out of the databricks namespace
// rather than the default hashicorp namespace (which has no databricks
// provider). Kept in lockstep with bundle/internal/tf/schema.ProviderSource
// but duplicated here to honour the ucm fork-divergence rule (no imports
// from bundle/**).
const ProviderSource = "databricks/databricks"

// ProviderVersion pins the databricks terraform provider version ucm
// renders into main.tf.json. Temporarily held one minor behind the DABs
// pin (1.113.0) because the internal terraform-proxy.dev.databricks.com
// mirror tops out at 1.112.0 at the time of writing. Bump back to match
// bundle/internal/tf/schema.ProviderVersion once the internal mirror
// catches up.
const ProviderVersion = "1.112.0"

// Terraform CLI override env vars. Same wire names as bundle/deploy/terraform
// so a user's DATABRICKS_TF_EXEC_PATH works for both subcommands.
const (
	ExecPathEnv        = "DATABRICKS_TF_EXEC_PATH"
	VersionEnv         = "DATABRICKS_TF_VERSION"
	CliConfigPathEnv   = "DATABRICKS_TF_CLI_CONFIG_FILE"
	ProviderVersionEnv = "DATABRICKS_TF_PROVIDER_VERSION"
)

// defaultTerraformVersion is the Terraform CLI version we download when the
// user does not override it. Kept in lockstep with bundle/deploy/terraform so
// the two subcommands use a single binary on disk.
var defaultTerraformVersion = version.Must(version.NewVersion("1.5.5"))

// GetTerraformVersion returns the terraform version to use, honouring
// DATABRICKS_TF_VERSION when set.
func GetTerraformVersion(ctx context.Context) (*version.Version, bool, error) {
	v, ok, err := lookupVersionFromEnv(ctx)
	if err != nil {
		return nil, false, err
	}
	if ok {
		return v, false, nil
	}
	return defaultTerraformVersion, true, nil
}
