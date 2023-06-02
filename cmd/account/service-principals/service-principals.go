// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package service_principals

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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
  
  Creates a new service principal in the Databricks account.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := a.ServicePrincipals.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq iam.DeleteAccountServicePrincipalRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a service principal.`,
	Long: `Delete a service principal.
  
  Delete a single service principal in the Databricks account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No ID argument specified. Loading names for Account Service Principals drop-down."
				names, err := a.ServicePrincipals.ServicePrincipalDisplayNameToIdMap(ctx, iam.ListAccountServicePrincipalsRequest{})
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "Unique ID for a service principal in the Databricks account")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have unique id for a service principal in the databricks account")
			}
			deleteReq.Id = args[0]
		}

		err = a.ServicePrincipals.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq iam.GetAccountServicePrincipalRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get ID",
	Short: `Get service principal details.`,
	Long: `Get service principal details.
  
  Gets the details for a single service principal define in the Databricks
  account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No ID argument specified. Loading names for Account Service Principals drop-down."
				names, err := a.ServicePrincipals.ServicePrincipalDisplayNameToIdMap(ctx, iam.ListAccountServicePrincipalsRequest{})
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "Unique ID for a service principal in the Databricks account")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have unique id for a service principal in the databricks account")
			}
			getReq.Id = args[0]
		}

		response, err := a.ServicePrincipals.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq iam.ListAccountServicePrincipalsRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
  
  Gets the set of service principals associated with a Databricks account.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := a.ServicePrincipals.ListAll(ctx, listReq)
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
	Use:   "patch ID",
	Short: `Update service principal details.`,
	Long: `Update service principal details.
  
  Partially updates the details of a single service principal in the Databricks
  account.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = patchJson.Unmarshal(&patchReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No ID argument specified. Loading names for Account Service Principals drop-down."
				names, err := a.ServicePrincipals.ServicePrincipalDisplayNameToIdMap(ctx, iam.ListAccountServicePrincipalsRequest{})
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "Unique ID for a service principal in the Databricks account")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have unique id for a service principal in the databricks account")
			}
			patchReq.Id = args[0]
		}

		err = a.ServicePrincipals.Patch(ctx, patchReq)
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
	Use:   "update ID",
	Short: `Replace service principal.`,
	Long: `Replace service principal.
  
  Updates the details of a single service principal.
  
  This action replaces the existing service principal with the same name.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No ID argument specified. Loading names for Account Service Principals drop-down."
				names, err := a.ServicePrincipals.ServicePrincipalDisplayNameToIdMap(ctx, iam.ListAccountServicePrincipalsRequest{})
				close(promptSpinner)
				if err != nil {
					return err
				}
				id, err := cmdio.Select(ctx, names, "Databricks service principal ID")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have databricks service principal id")
			}
			updateReq.Id = args[0]
		}

		err = a.ServicePrincipals.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service AccountServicePrincipals
