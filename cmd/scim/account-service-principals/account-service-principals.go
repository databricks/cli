package account_service_principals

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "account-service-principals",
	Short: `Identities for use with jobs, automated tools, and systems such as scripts, apps, and CI/CD platforms.`,
}

var createReq scim.ServicePrincipal

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.Active, "active", false, `If this user is active.`)
	createCmd.Flags().StringVar(&createReq.ApplicationId, "application-id", "", `UUID relating to the service principal.`)
	createCmd.Flags().StringVar(&createReq.DisplayName, "display-name", "", `String that represents a concatenation of given and family names.`)
	// TODO: array: entitlements
	createCmd.Flags().StringVar(&createReq.ExternalId, "external-id", "", ``)
	// TODO: array: groups
	createCmd.Flags().StringVar(&createReq.Id, "id", "", `Databricks service principal ID.`)
	// TODO: array: roles

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a service principal.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.ServicePrincipals.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq scim.DeleteServicePrincipalRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", "", `Unique ID for a service principal in the Databricks Account.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a service principal.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.ServicePrincipals.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq scim.GetServicePrincipalRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Id, "id", "", `Unique ID for a service principal in the Databricks Account.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get service principal details.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.ServicePrincipals.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var listReq scim.ListServicePrincipalsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.Attributes, "attributes", "", `Comma-separated list of attributes to return in response.`)
	listCmd.Flags().IntVar(&listReq.Count, "count", 0, `Desired number of results per page.`)
	listCmd.Flags().StringVar(&listReq.ExcludedAttributes, "excluded-attributes", "", `Comma-separated list of attributes to exclude in response.`)
	listCmd.Flags().StringVar(&listReq.Filter, "filter", "", `Query by which the results have to be filtered.`)
	listCmd.Flags().StringVar(&listReq.SortBy, "sort-by", "", `Attribute to sort the results.`)
	listCmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order to sort the results.`)
	listCmd.Flags().IntVar(&listReq.StartIndex, "start-index", 0, `Specifies the index of the first result.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List service principals.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.ServicePrincipals.ListAll(ctx, listReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var patchReq scim.PartialUpdate

func init() {
	Cmd.AddCommand(patchCmd)
	// TODO: short flags

	patchCmd.Flags().StringVar(&patchReq.Id, "id", "", `Unique ID for a group in the Databricks Account.`)
	// TODO: array: operations

}

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: `Update service principal details.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.ServicePrincipals.Patch(ctx, patchReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq scim.ServicePrincipal

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().BoolVar(&updateReq.Active, "active", false, `If this user is active.`)
	updateCmd.Flags().StringVar(&updateReq.ApplicationId, "application-id", "", `UUID relating to the service principal.`)
	updateCmd.Flags().StringVar(&updateReq.DisplayName, "display-name", "", `String that represents a concatenation of given and family names.`)
	// TODO: array: entitlements
	updateCmd.Flags().StringVar(&updateReq.ExternalId, "external-id", "", ``)
	// TODO: array: groups
	updateCmd.Flags().StringVar(&updateReq.Id, "id", "", `Databricks service principal ID.`)
	// TODO: array: roles

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Replace service principal.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.ServicePrincipals.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
