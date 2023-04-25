// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principals

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "service-principals",
	Short: `Identities for use with jobs, automated tools, and systems such as scripts, apps, and CI/CD platforms.`,
	Long: `Identities for use with jobs, automated tools, and systems such as scripts,
  apps, and CI/CD platforms. Databricks recommends creating service principals
  to run production jobs or modify production data. If all processes that act on
  production data run with service principals, interactive users do not need any
  write, delete, or modify privileges in production. This eliminates the risk of
  a user overwriting production data by accident.`,
}

// start create command

var createReq iam.ServicePrincipal
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().BoolVar(&createReq.Active, "active", createReq.Active, `If this user is active.`)
	createCmd.Flags().StringVar(&createReq.ApplicationId, "application-id", createReq.ApplicationId, `UUID relating to the service principal.`)
	createCmd.Flags().StringVar(&createReq.DisplayName, "display-name", createReq.DisplayName, `String that represents a concatenation of given and family names.`)
	// TODO: array: entitlements
	createCmd.Flags().StringVar(&createReq.ExternalId, "external-id", createReq.ExternalId, ``)
	// TODO: array: groups
	createCmd.Flags().StringVar(&createReq.Id, "id", createReq.Id, `Databricks service principal ID.`)
	// TODO: array: roles

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a service principal.`,
	Long: `Create a service principal.
  
  Creates a new service principal in the Databricks Workspace.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Id = args[0]

		response, err := w.ServicePrincipals.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq iam.DeleteServicePrincipalRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a service principal.`,
	Long: `Delete a service principal.
  
  Delete a single service principal in the Databricks Workspace.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.ServicePrincipals.ServicePrincipalDisplayNameToIdMap(ctx, iam.ListServicePrincipalsRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Unique ID for a service principal in the Databricks Workspace")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a service principal in the databricks workspace")
		}
		deleteReq.Id = args[0]

		err = w.ServicePrincipals.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq iam.GetServicePrincipalRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get ID",
	Short: `Get service principal details.`,
	Long: `Get service principal details.
  
  Gets the details for a single service principal define in the Databricks
  Workspace.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.ServicePrincipals.ServicePrincipalDisplayNameToIdMap(ctx, iam.ListServicePrincipalsRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "Unique ID for a service principal in the Databricks Workspace")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have unique id for a service principal in the databricks workspace")
		}
		getReq.Id = args[0]

		response, err := w.ServicePrincipals.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq iam.ListServicePrincipalsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.Attributes, "attributes", listReq.Attributes, `Comma-separated list of attributes to return in response.`)
	listCmd.Flags().IntVar(&listReq.Count, "count", listReq.Count, `Desired number of results per page.`)
	listCmd.Flags().StringVar(&listReq.ExcludedAttributes, "excluded-attributes", listReq.ExcludedAttributes, `Comma-separated list of attributes to exclude in response.`)
	listCmd.Flags().StringVar(&listReq.Filter, "filter", listReq.Filter, `Query by which the results have to be filtered.`)
	listCmd.Flags().StringVar(&listReq.SortBy, "sort-by", listReq.SortBy, `Attribute to sort the results.`)
	listCmd.Flags().Var(&listReq.SortOrder, "sort-order", `The order to sort the results.`)
	listCmd.Flags().IntVar(&listReq.StartIndex, "start-index", listReq.StartIndex, `Specifies the index of the first result.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List service principals.`,
	Long: `List service principals.
  
  Gets the set of service principals associated with a Databricks Workspace.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.ServicePrincipals.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start patch command

var patchReq iam.PartialUpdate
var patchJson flags.JsonFlag

func init() {
	Cmd.AddCommand(patchCmd)
	// TODO: short flags
	patchCmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: operations

}

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: `Update service principal details.`,
	Long: `Update service principal details.
  
  Partially updates the details of a single service principal in the Databricks
  Workspace.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = patchJson.Unmarshall(&patchReq)
		if err != nil {
			return err
		}
		patchReq.Id = args[0]

		err = w.ServicePrincipals.Patch(ctx, patchReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq iam.ServicePrincipal
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().BoolVar(&updateReq.Active, "active", updateReq.Active, `If this user is active.`)
	updateCmd.Flags().StringVar(&updateReq.ApplicationId, "application-id", updateReq.ApplicationId, `UUID relating to the service principal.`)
	updateCmd.Flags().StringVar(&updateReq.DisplayName, "display-name", updateReq.DisplayName, `String that represents a concatenation of given and family names.`)
	// TODO: array: entitlements
	updateCmd.Flags().StringVar(&updateReq.ExternalId, "external-id", updateReq.ExternalId, ``)
	// TODO: array: groups
	updateCmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Databricks service principal ID.`)
	// TODO: array: roles

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Replace service principal.`,
	Long: `Replace service principal.
  
  Updates the details of a single service principal.
  
  This action replaces the existing service principal with the same name.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.Id = args[0]

		err = w.ServicePrincipals.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service ServicePrincipals
