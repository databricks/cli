package generate

import (
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// NewGenerateStorageCredentialCommand returns the `ucm generate storage-credential` cobra subcommand.
func NewGenerateStorageCredentialCommand() *cobra.Command {
	var existingStorageCredentialName string
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "storage-credential",
		Short: "Generate ucm configuration for an existing Unity Catalog storage credential",
		Long: `Generate ucm configuration for an existing Unity Catalog storage credential.

Fetches the storage credential by name and writes a per-resource YAML
fragment to --output-dir that you can include from your ucm.yml.

Note: Azure service-principal credentials carry a ClientSecret that UC
does not echo back. The generated YAML carries an empty placeholder; fill
it in by hand before the next ucm deploy or the credential will be
recreated with an empty secret.

Example:
  databricks ucm generate storage-credential --existing-storage-credential-name prod_cred --key prod_cred`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.Flags().StringVar(&existingStorageCredentialName, "existing-storage-credential-name", "", "Name of the existing storage credential to import.")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write the generated configuration into.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files in --output-dir.")
	_ = cmd.MarkFlagRequired("existing-storage-credential-name")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		if w == nil {
			return fmt.Errorf("workspace client not configured")
		}

		info, err := w.StorageCredentials.GetByName(ctx, existingStorageCredentialName)
		if err != nil {
			return fmt.Errorf("fetch storage credential %q: %w", existingStorageCredentialName, err)
		}

		res, warn := convertStorageCredential(info)
		if res == nil {
			return fmt.Errorf("storage credential %q has an unsupported identity type", info.Name)
		}
		if warn != "" {
			cmdio.LogString(ctx, "warning: "+warn)
		}

		key := getKey(cmd, info.Name)
		outPath, err := writeResourceYAML(outputDir, "storage_credentials", key, res, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Wrote storage credential %q to %s", key, filepath.ToSlash(outPath)))
		return nil
	}

	return cmd
}

// convertStorageCredential maps a server-side StorageCredentialInfo onto the
// ucm StorageCredential resource using a direct SDK call (the per-kind
// generate workflow that replaced the bulk-scan engine). Returns (resource,
// warning). The warning is surfaced verbatim — it is informational, not fatal.
func convertStorageCredential(info *catalog.StorageCredentialInfo) (*resources.StorageCredential, string) {
	res := &resources.StorageCredential{
		CreateStorageCredential: catalog.CreateStorageCredential{
			Name:     info.Name,
			Comment:  info.Comment,
			ReadOnly: info.ReadOnly,
		},
	}
	var warn string
	switch {
	case info.AwsIamRole != nil:
		res.AwsIamRole = &catalog.AwsIamRoleRequest{RoleArn: info.AwsIamRole.RoleArn}
	case info.AzureManagedIdentity != nil:
		res.AzureManagedIdentity = &catalog.AzureManagedIdentityRequest{
			AccessConnectorId: info.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: info.AzureManagedIdentity.ManagedIdentityId,
		}
	case info.AzureServicePrincipal != nil:
		res.AzureServicePrincipal = &catalog.AzureServicePrincipal{
			DirectoryId:   info.AzureServicePrincipal.DirectoryId,
			ApplicationId: info.AzureServicePrincipal.ApplicationId,
			ClientSecret:  "", // UC does not echo the secret; user must fill it in.
		}
		warn = fmt.Sprintf("storage_credential %q: azure_service_principal.client_secret not available from UC; set it in the generated yaml before the next deploy", info.Name)
	case info.DatabricksGcpServiceAccount != nil:
		res.DatabricksGcpServiceAccount = &catalog.DatabricksGcpServiceAccountRequest{}
	default:
		return nil, ""
	}
	return res, warn
}
